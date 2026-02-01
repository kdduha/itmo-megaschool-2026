package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server      ServerConfig
	OpenAI      OpenAIConfig
	RedisConfig RedisConfig
	CacheEnable bool `env:"CACHE_ENABLE"`
}

type RedisConfig struct {
	Addr     string        `env:"REDIS_ADDR" env-default:"redis:6379"`
	Password string        `env:"REDIS_PASSWORD"`
	DB       int           `env:"REDIS_DB" env-default:"0"`
	TTL      time.Duration `env:"REDIS_TTL" env-default:"10m"`
}

type ServerConfig struct {
	Port            string        `env:"SERVER_PORT" envDefault:"8080"`
	Timeout         time.Duration `env:"SERVER_TIMEOUT" envDefault:"2m"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" envDefault:"10s"`
	ThrottleLimit   int           `env:"SERVER_THROTTLE_LIMIT" envDefault:"50"`
}

type OpenAIConfig struct {
	APIKey  string `env:"OPENAI_API_KEY"`
	BaseURL string `env:"OPENAI_BASE_URL" envDefault:"http://localhost:8000/v1"`
	Model   string `env:"OPENAI_MODEL" envDefault:"default"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
