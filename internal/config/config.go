package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Webhook  WebhookConfig
}

type ServerConfig struct {
	Port         string        `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"10s"`
}

type PostgresConfig struct {
	DSN             string        `envconfig:"POSTGRES_DSN" required:"true"`
	MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"POSTGRES_CONN_MAX_LIFETIME" default:"5m"`
}

type RedisConfig struct {
	Addr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:""`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

type WorkerConfig struct {
	Concurrency      int           `envconfig:"WORKER_CONCURRENCY" default:"10"`
	RateLimitPerSec  int           `envconfig:"WORKER_RATE_LIMIT_PER_SEC" default:"100"`
	MaxRetries       int           `envconfig:"WORKER_MAX_RETRIES" default:"3"`
	RetryBaseDelay   time.Duration `envconfig:"WORKER_RETRY_BASE_DELAY" default:"5s"`
	ArchiveAfter     time.Duration `envconfig:"ARCHIVE_AFTER" default:"720h"`
	ArchiveBatchSize int           `envconfig:"ARCHIVE_BATCH_SIZE" default:"1000"`
}

type WebhookConfig struct {
	URL     string        `envconfig:"WEBHOOK_URL" required:"true"`
	Timeout time.Duration `envconfig:"WEBHOOK_TIMEOUT" default:"10s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
