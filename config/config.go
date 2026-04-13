package config

import (
	"log"
	"os"
	"strings"
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
	v.SetConfigName(env)
	v.SetConfigType("json")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.AutomaticEnv()

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

	// Environment overrides used by Docker Compose and deployments.
	if appPort := os.Getenv("APP_PORT"); appPort != "" {
		cfg.App.Port = appPort
	}
	if appEnv := os.Getenv("APP_ENV"); appEnv != "" {
		cfg.App.Env = appEnv
	}

	if esAddresses := os.Getenv("ES_ADDRESSES"); esAddresses != "" {
		parts := strings.Split(esAddresses, ",")
		addresses := make([]string, 0, len(parts))
		for _, p := range parts {
			if addr := strings.TrimSpace(p); addr != "" {
				addresses = append(addresses, addr)
			}
		}
		if len(addresses) > 0 {
			cfg.Elasticsearch.Addresses = addresses
		}
	} else if esURL := os.Getenv("ES_URL"); esURL != "" {
		cfg.Elasticsearch.Addresses = []string{esURL}
	}

	if esUser := os.Getenv("ES_USERNAME"); esUser != "" {
		cfg.Elasticsearch.Username = esUser
	}
	if esPass := os.Getenv("ES_PASSWORD"); esPass != "" {
		cfg.Elasticsearch.Password = esPass
	}
	if esAPIKey := os.Getenv("ES_API_KEY"); esAPIKey != "" {
		cfg.Elasticsearch.APIKey = esAPIKey
	}
	if esCloudID := os.Getenv("ES_CLOUD_ID"); esCloudID != "" {
		cfg.Elasticsearch.CloudID = esCloudID
	}
	if esCACert := os.Getenv("ES_CA_CERT"); esCACert != "" {
		cfg.Elasticsearch.CACert = esCACert
	}
	if cors := os.Getenv("CORS_ALLOWED_ORIGINS"); cors != "" {
		cfg.CORSAllowedOrigins = cors
	}

	if cfg.App.Env == "" {
		cfg.App.Env = env
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
