package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

type ServerConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Playground bool   `mapstructure:"playground"`
}

type StorageConfig struct {
	Type string `mapstructure:"type"`
}

type PostgresConfig struct {
	DSN string `mapstructure:"dsn"`
}

type LoggerConfig struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("OZON_TASK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	bindEnv(v)

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.playground", true)
	v.SetDefault("storage.type", "memory")
	v.SetDefault("logger.level", "debug")
	v.SetDefault("logger.development", true)
}

func bindEnv(v *viper.Viper) {
	keys := []string{
		"server.host",
		"server.port",
		"server.playground",
		"storage.type",
		"postgres.dsn",
		"logger.level",
		"logger.development",
	}

	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}
