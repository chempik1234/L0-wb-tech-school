package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"order_service/pkg/kafka"
	"order_service/pkg/postgres"
)

// OrderServiceConfig is named after the microservice, not the service struct!
type OrderServiceConfig struct {
	KafkaTopic   string `yaml:"kafka_topic" env-prefix:"KAFKA_TOPIC"`
	KafkaGroupID string `yaml:"kafka_group_id" env-prefix:"KAFKA_GROUP"`
	HTTPPort     int    `yaml:"http-port" env:"HTTP_PORT" env-default:"8080"`
}

type Config struct {
	OrderService OrderServiceConfig `yaml:"microservice" env-prefix:"ORDER_SERVICE_"`
	Kafka        kafka.Config       `yaml:"kafka" env-prefix:"KAFKA_"`
	Postgres     postgres.Config    `yaml:"postgres" env-prefix:"POSTGRES_"`
}

func TryRead() (Config, error) {
	var cfg Config
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("warning: failed to read .env: %s", err)
	}
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		return Config{},
			fmt.Errorf("failed to read env variables after accessing .env: %w", err)
	}
	return cfg, nil
}
