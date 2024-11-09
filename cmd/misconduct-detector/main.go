package main

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/observability"
	"github.com/TicketsBot/common/rpc"
	"github.com/TicketsBot/misconduct-detector/internal/config"
	"github.com/TicketsBot/misconduct-detector/internal/processor"
	"github.com/TicketsBot/misconduct-detector/internal/processor/rules"
	"github.com/TicketsBot/misconduct-detector/internal/queue"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rxdn/gdl/cache"
	"github.com/rxdn/gdl/objects/guild"
	"github.com/rxdn/gdl/rest/request"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	config := must(config.LoadFromEnv())

	// Build logger
	if config.SentryDsn != nil {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: *config.SentryDsn,
		}); err != nil {
			panic(fmt.Errorf("sentry.Init: %w", err))
		}
	}

	var logger *zap.Logger
	var err error
	if config.JsonLogs {
		loggerConfig := zap.NewProductionConfig()
		loggerConfig.Level.SetLevel(config.LogLevel)

		logger, err = loggerConfig.Build(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
			zap.WrapCore(observability.ZapSentryAdapter(observability.EnvironmentProduction)),
		)
	} else {
		loggerConfig := zap.NewDevelopmentConfig()
		loggerConfig.Level.SetLevel(config.LogLevel)
		loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		logger, err = loggerConfig.Build(zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	}

	if err != nil {
		panic(fmt.Errorf("failed to initialise zap logger: %w", err))
	}

	if config.PrometheusServerAddr != nil {
		logger.Info("Starting Prometheus server", zap.String("addr", *config.PrometheusServerAddr))

		http.Handle("/metrics", promhttp.Handler())
		go func() {
			if err := http.ListenAndServe(*config.PrometheusServerAddr, nil); err != nil {
				panic(err)
			}
		}()
	}

	// Build app context
	appContext := must(buildAppContext(config, logger))

	if config.Discord.ProxyUrl != nil {
		logger.Info("Using proxy", zap.String("url", *config.Discord.ProxyUrl))
		request.RegisterPreRequestHook(func(_ string, req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = *config.Discord.ProxyUrl
		})
	}

	// Connect to Kafka
	guildCh := make(chan guild.Guild, 10)
	consumer := queue.NewConsumer(config, logger.With(zap.String("module", "consumer")), guildCh)

	logger.Info("Starting RPC client")
	rpcClient, err := rpc.NewClient(
		logger.With(zap.String("service", "rpc-client")),
		rpc.Config{
			Brokers:             config.Kafka.Brokers,
			ConsumerConcurrency: 1,
		},
		map[string]rpc.Listener{
			config.Kafka.EventsTopic: consumer,
		},
	)
	if err != nil {
		logger.Fatal("Failed to start RPC client", zap.Error(err))
		return
	}

	logger.Info("RPC client started")

	wg := &sync.WaitGroup{}
	delegators := make([]*processor.Delegator, config.ConcurrentTasks)
	for i := 0; i < config.ConcurrentTasks; i++ {
		wg.Add(1)

		delegator := processor.NewDelegator(
			config,
			logger.With(zap.String("module", "delegator")),
			appContext,
			rules.Ruleset,
			queue.NewKafkaProducer(config, rpcClient),
			guildCh,
		)

		go func() {
			defer wg.Done()
			delegator.Run()
		}()

		delegators[i] = delegator
	}

	awaitShutdown(logger, wg, delegators)
}

func awaitShutdown(logger *zap.Logger, wg *sync.WaitGroup, delegators []*processor.Delegator) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done

	logger.Info("Shutting down")
	for _, delegator := range delegators {
		delegator.Shutdown()
	}

	wgCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgCh)
	}()

	select {
	case <-wgCh:
		logger.Info("All delegators have shut down")
	case <-time.After(10 * time.Second):
		logger.Warn("Some delegators have not shut down, but timeout has expired")
	}
}

func buildAppContext(config config.Config, logger *zap.Logger) (*rules.AppContext, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger.Info("Connecting to cache")
	pool, err := pgxpool.Connect(ctx, config.Cache.Uri)
	if err != nil {
		return nil, err
	}

	cache := cache.NewPgCache(pool, cache.CacheOptions{
		Guilds:   true,
		Users:    true,
		Members:  true,
		Channels: true,
	})
	logger.Info("Connected to cache")

	return rules.NewAppContext(config, &cache), nil
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}
