package simulator_service

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HttpPort string

	KafkaTopic             string
	KafkaHost              string
	KafkaPort              uint16
	KafkaBrokers           []string
	KafkaNumPartitions     int
	KafkaReplicationFactor int
}

func ConfigFromEnv() *Config {
	kafkaPort, _ := strconv.ParseUint(os.Getenv("KAFKA_PORT"), 10, 16)
	numPartitions, _ := strconv.Atoi(os.Getenv("KAFKA_NUM_PARTITIONS"))
	replicationFactor, _ := strconv.Atoi(os.Getenv("KAFKA_REPLICATION_FACTOR"))
	return &Config{
		HttpPort:               os.Getenv("SIMULATOR_SERVICE_HTTP_PORT"),
		KafkaTopic:             os.Getenv("SIMULATOR_SERVICE_KAFKA_TOPIC"),
		KafkaHost:              os.Getenv("KAFKA_HOST"),
		KafkaPort:              uint16(kafkaPort),
		KafkaBrokers:           strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		KafkaNumPartitions:     numPartitions,
		KafkaReplicationFactor: replicationFactor,
	}
}
