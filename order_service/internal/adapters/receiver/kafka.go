package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"order_service/internal/models"
)

type KafkaReceiver struct {
	reader *kafka.Reader
}

func NewKafkaReceiver(reader *kafka.Reader) *KafkaReceiver {
	return &KafkaReceiver{
		reader: reader,
	}
}

func (k *KafkaReceiver) Consume(ctx context.Context) (models.Order, error) {
	msg, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return models.Order{}, fmt.Errorf("error while reading from kafka: %w", err)
	}
	var order models.Order
	err = json.Unmarshal(msg.Value, &order)
	if err != nil {
		return models.Order{}, fmt.Errorf("error while unmarshalling message: %w", err)
	}
	return order, nil
}
