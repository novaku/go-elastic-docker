package config

import (
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	App                AppConfig `mapstructure:"app"`
	Elasticsearch      ESConfig  `mapstructure:"elasticsearch"`
	CORSAllowedOrigins string    `mapstructure:"cors_allowed_origins"`
}

type AppConfig struct {
	Port         string        `mapstructure:"port"`
	Env          string        `mapstructure:"env"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type ESConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
	APIKey    string   `mapstructure:"api_key"`
	CACert    string   `mapstructure:"ca_cert"`
	CloudID   string   `mapstructure:"cloud_id"`
}

// Load loads configuration from JSON file using Viper
func Load() *Config {
	v := viper.New()
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}
	if env != "local" && env != "production" {
		log.Printf("unknown APP_ENV=%q, fallback to local", env)
		env = "local"
	}
	v.SetConfigName(env)
	v.SetConfigType("json")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("loading config with viper: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}

	// Backward compatibility: support legacy `elasticsearch.url` key.
	if len(cfg.Elasticsearch.Addresses) == 0 {
		if url := v.GetString("elasticsearch.url"); url != "" {
			cfg.Elasticsearch.Addresses = []string{url}
		}
	}

	if cfg.App.Env == "" {
		cfg.App.Env = env
	}

	// If duration values are provided as raw integers in config files,
	// Viper unmarshals them as nanoseconds. Promote very small values to seconds.
	if cfg.App.ReadTimeout > 0 && cfg.App.ReadTimeout < time.Second {
		cfg.App.ReadTimeout = cfg.App.ReadTimeout * time.Second
	}
	if cfg.App.WriteTimeout > 0 && cfg.App.WriteTimeout < time.Second {
		cfg.App.WriteTimeout = cfg.App.WriteTimeout * time.Second
	}

	if cfg.App.ReadTimeout <= 0 {
		cfg.App.ReadTimeout = 10 * time.Second
	}
	if cfg.App.WriteTimeout <= 0 {
		cfg.App.WriteTimeout = 10 * time.Second
	}

	return &cfg
}

// BuildLogger returns a production-grade logger in production and a
// human-friendly development logger otherwise.
func (c *Config) BuildLogger() *zap.Logger {
	var cfg zap.Config
	if c.App.Env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("building logger: " + err.Error())
	}
	return logger
}
