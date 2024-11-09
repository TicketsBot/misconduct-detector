package processor

import (
	"context"
	"github.com/TicketsBot/common/rpc/model"
	"github.com/TicketsBot/misconduct-detector/internal/config"
	"github.com/TicketsBot/misconduct-detector/internal/processor/rules"
	"github.com/TicketsBot/misconduct-detector/internal/queue"
	"github.com/rxdn/gdl/objects/guild"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type Delegator struct {
	config     config.Config
	logger     *zap.Logger
	appContext *rules.AppContext

	evaluators []rules.Evaluator

	producer   queue.Producer
	ch         <-chan guild.Guild
	shutdownCh chan struct{}
}

const ActionThreshold = 25

func NewDelegator(
	config config.Config,
	logger *zap.Logger,
	appContext *rules.AppContext,
	evaluators []rules.Evaluator,
	producer queue.Producer,
	ch <-chan guild.Guild,
) *Delegator {
	return &Delegator{
		config:     config,
		logger:     logger,
		appContext: appContext,
		evaluators: evaluators,
		producer:   producer,
		ch:         ch,
		shutdownCh: make(chan struct{}),
	}
}

func (d *Delegator) Run() {
	for {
		select {
		case guild := <-d.ch:
			d.handleGuild(&guild)
		case <-d.shutdownCh:
			d.logger.Info("Shutting down delegator")
			return
		}
	}
}

func (d *Delegator) Shutdown() {
	close(d.shutdownCh)
}

func (d *Delegator) handleGuild(guild *guild.Guild) {
	logger := d.logger.With(zap.Uint64("guild_id", guild.Id))
	logger.Info("Evaluating guild")

	ctx, cancel := context.WithTimeout(context.Background(), d.config.TaskTimeout)
	defer cancel()

	scores := make(map[string]int)
	mu := &sync.Mutex{}

	var group errgroup.Group
	for _, evaluator := range d.evaluators {
		properties := evaluator.Properties()

		ruleEvaluationCounter.WithLabelValues(properties.RuleName).Inc()

		if properties.ShouldSpawnGoroutine {
			evaluator := evaluator
			group.Go(func() error {
				now := time.Now()
				defer func() {
					ruleExecutionTimeHistogram.WithLabelValues(properties.RuleName).Observe(time.Since(now).Seconds())
				}()

				s, err := evaluator.Evaluate(ctx, d.appContext, guild)
				if err != nil {
					return err
				}

				mu.Lock()
				scores[properties.RuleName] = s
				mu.Unlock()

				ruleScoreCounter.WithLabelValues(properties.RuleName).Add(float64(s))
				return nil
			})
		} else {
			now := time.Now()
			s, err := evaluator.Evaluate(ctx, d.appContext, guild)
			ruleExecutionTimeHistogram.WithLabelValues(properties.RuleName).Observe(time.Since(now).Seconds())
			if err != nil {
				d.logger.Error("Failed to evaluate guild", zap.Error(err))
				continue
			}

			mu.Lock()
			scores[properties.RuleName] = s
			mu.Unlock()
		}
	}

	if err := group.Wait(); err != nil {
		logger.Error("Failed to evaluate guild", zap.Error(err))
		// Don't return! If the score still exceeds the threshold, we should still take action
	}

	var totalScore int
	for _, score := range scores {
		totalScore += score
	}

	if totalScore > 100 {
		totalScore = 100
	}

	d.logger.Debug("Evaluated guild", zap.Int("total_score", totalScore), zap.Any("scores", scores))

	if totalScore > ActionThreshold {
		d.logger.Info("Guild is above alert threshold, publishing", zap.Int("total_score", totalScore), zap.Any("scores", scores))

		if err := d.producer.PublishAlert(ctx, model.MisconductAlert{
			Guild:      guild,
			Score:      totalScore,
			RuleScores: scores,
		}); err != nil {
			logger.Error("Failed to publish alert", zap.Error(err))
		}
	}
}
