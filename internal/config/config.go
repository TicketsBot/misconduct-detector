package config

import (
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap/zapcore"
	"time"
)

type Config struct {
	SentryDsn       *string       `env:"SENTRY_DSN"`
	JsonLogs        bool          `env:"JSON_LOGS"`
	LogLevel        zapcore.Level `env:"LOG_LEVEL" envDefault:"info"`
	TaskTimeout     time.Duration `env:"TASK_TIMEOUT" envDefault:"10s"`
	ConcurrentTasks int           `env:"CONCURRENT_TASKS" envDefault:"3"`

	Kafka struct {
		Brokers        []string `env:"BROKERS,required" envSeparator:","`
		EventsTopic    string   `env:"EVENTS_TOPIC,required"`
		DetectionTopic string   `env:"DETECTION_TOPIC,required"`
	} `envPrefix:"KAFKA_"`

	Discord struct {
		ProxyUrl *string `env:"PROXY_URL"`
		Token    string  `env:"TOKEN,required"`
	} `envPrefix:"DISCORD_"`

	Cache struct {
		Uri string `env:"URI,required"`
	} `envPrefix:"CACHE_"`
}

func LoadFromEnv() (Config, error) {
	return env.ParseAs[Config]()
}
