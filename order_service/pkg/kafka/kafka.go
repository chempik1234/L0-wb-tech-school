package kafka

import (
	"context"
	"fmt"
	"github.com/go-faster/errors"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"order_service/pkg/logger"
	"time"
)

// Config is kafka only config without certain properties for a service
type Config struct {
	Host    string   `yaml:"host" env:"HOST" env-default:"kafka"`
	Port    uint16   `yaml:"port" env:"PORT" env-default:"9092"`
	Brokers []string `yaml:"brokers" env:"BROKERS" env-separator:","`

	MinBytes       int `yaml:"min_bytes" env:"MIN_BYTES" env-default:"10"`
	MaxBytes       int `yaml:"max_bytes" env:"MAX_BYTES" env-default:"1048576"` // 1MB
	MaxWaitMs      int `yaml:"max_wait_ms" env:"MAX_WAIT_MS" env-default:"500"`
	CommitInterval int `yaml:"commit_interval_ms" env:"COMMIT_INTERVAL_MS" env-default:"1000"`

	NumPartitions     int `yaml:"num_partitions" env:"NUM_PARTITIONS" env-default:"1"`
	ReplicationFactor int `yaml:"replication_factor" env:"REPLICATION_FACTOR" env-default:"1"`
}

// NewReader creates a new kafka.Reader with given settings
func NewReader(ctx context.Context, cfg Config, topic, groupID string) *kafka.Reader {
	l := logger.GetOrCreateLoggerFromCtx(ctx)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        time.Duration(cfg.MaxWaitMs) * time.Millisecond,
		CommitInterval: time.Duration(cfg.CommitInterval) * time.Millisecond,
	})
	l.Info(ctx, "connected to Kafka topic",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", topic),
		zap.String("group_id", groupID),
	)
	return r
}

// CreateTopicIfNotExists safely creates a topic. Supposed to be called on startup to ensure that topic exists
func CreateTopicIfNotExists(cfg Config, topic string, numPartitions, replicationFactor int) error {
	if topic == "" {
		return errors.New("topic name mustn't be empty")
	}

	conn, err := kafka.Dial("tcp", cfg.Brokers[0])
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
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	})
}

// CreateTopicWithRetry safely creates a topic using CreateTopicIfNotExists, but it gives a few tries
func CreateTopicWithRetry(cfg Config, topic string, numPartitions, replicationFactor int, retries int) error {
	var err error
	for i := 0; i < retries; i++ {
		err = CreateTopicIfNotExists(cfg, topic, numPartitions, replicationFactor)
		if err == nil {
			return nil
		}

		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		time.Sleep(time.Second * time.Duration(i))
	}
	return err
}
