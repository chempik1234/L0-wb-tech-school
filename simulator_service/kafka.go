package simulator_service

import (
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
)

func NewWriter(cfg *Config) *kafka.Writer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        cfg.KafkaTopic,
		RequiredAcks: kafka.RequireAll,
		Balancer:     &kafka.LeastBytes{},
		Async:        false,
	}
	return w
}

func CreateTopicIfNotExists(cfg *Config) error {
	if cfg.KafkaTopic == "" {
		return errors.New("topic name mustn't be empty")
	}

	conn, err := kafka.Dial("tcp", cfg.KafkaBrokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp",
		fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return err
	}

	defer controllerConn.Close()

	return controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.KafkaTopic,
		NumPartitions:     cfg.KafkaNumPartitions,
		ReplicationFactor: cfg.KafkaReplicationFactor,
	})
}
