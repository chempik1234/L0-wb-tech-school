package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/pkg/logger"
)

type KafkaReceiver struct {
	reader *kafka.Reader
	// dcq    []kafka.Message
}

func NewKafkaReceiver(reader *kafka.Reader) ports.OrderReceiver[kafka.Message] {
	return &KafkaReceiver{
		reader: reader,
		// dcq: make([]kafka.Message),
	}
}

func (k *KafkaReceiver) Consume(ctx context.Context) (models.Order, kafka.Message, error) {
	msg, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return models.Order{}, kafka.Message{}, fmt.Errorf("error while reading from kafka: %w", err)
	}
	var order models.Order
	err = json.Unmarshal(msg.Value, &order)
	if err != nil {
		return models.Order{}, kafka.Message{}, fmt.Errorf("error while unmarshalling message: %w", err)
	}
	return order, kafka.Message(msg), nil
}

func (k *KafkaReceiver) OnSuccess(ctx context.Context, givenMessage kafka.Message) error {
	return k.reader.CommitMessages(ctx, givenMessage)
}

func (k *KafkaReceiver) OnFail(ctx context.Context, givenMessage kafka.Message) error {
	//k.sendToDLQ(givenMessage)
	logger.GetLoggerFromCtx(ctx).Error(ctx, "failed processing a message: %w")
	return nil
}

func (k *KafkaReceiver) sendToDLQ(givenMessage kafka.Message) {
	// TODO: DLQ with max retries
	panic("implement me")
}
