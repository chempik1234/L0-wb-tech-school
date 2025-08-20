package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"order_service/pkg/kafka"
	"order_service/pkg/postgres"
)

// OrderServiceConfig is named after the microservice, not the service struct!
type OrderServiceConfig struct {
	KafkaTopic    string `yaml:"kafka_topic" env:"KAFKA_TOPIC"`
	KafkaGroupID  string `yaml:"kafka_group_id" env:"KAFKA_GROUP_ID"`
	HTTPPort      int    `yaml:"http_port" env:"HTTP_PORT"`
	CacheCapacity int    `yaml:"cache_capacity" env:"CACHE_CAPACITY"`

	MaxSaveRetriesAmount   int `yaml:"max_save_retries_amount" env:"MAX_SAVE_RETRIES_AMOUNT"`
	MaxSaveRetriesCapacity int `yaml:"max_save_retries_capacity" env:"MAX_SAVE_RETRIES_CAPACITY"`
	SaveBackoffSeconds     int `yaml:"save_backoff_seconds" env:"SAVE_BACKOFF_SECONDS"`
}

type Config struct {
	OrderService OrderServiceConfig `yaml:"order_service" env-prefix:"ORDER_SERVICE_"`
	Kafka        kafka.Config       `yaml:"kafka" env-prefix:"KAFKA_"`
	Postgres     postgres.Config    `yaml:"postgres" env-prefix:"POSTGRES_"`
}

func TryRead() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{},
			fmt.Errorf("failed to read env variables after accessing .env: %w", err)
	}
	return cfg, nil
}
