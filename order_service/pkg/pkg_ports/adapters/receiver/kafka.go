package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"order_service/pkg/logger"
	"order_service/pkg/pkg_ports"
)

type KafkaReceiver[Value any] struct {
	reader *kafka.Reader
	// dcq    []kafka.Message
}

func NewKafkaReceiver[ValueType any](reader *kafka.Reader) pkg_ports.Receiver[ValueType, kafka.Message] {
	return &KafkaReceiver[ValueType]{
		reader: reader,
		// dcq: make([]kafka.Message),
	}
}

func (k *KafkaReceiver[ValueType]) Consume(ctx context.Context) (ValueType, kafka.Message, error) {
	msg, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return *new(ValueType), kafka.Message{}, fmt.Errorf("error while reading from kafka: %w", err)
	}
	var value ValueType
	err = json.Unmarshal(msg.Value, &value)
	if err != nil {
		return *new(ValueType), kafka.Message{}, fmt.Errorf("error while unmarshalling message: %w", err)
	}
	return value, msg, nil
}

func (k *KafkaReceiver[_]) OnSuccess(ctx context.Context, givenMessage kafka.Message) error {
	return k.reader.CommitMessages(ctx, givenMessage)
}

func (k *KafkaReceiver[_]) OnFail(ctx context.Context, givenMessage kafka.Message) error {
	//k.sendToDLQ(givenMessage)
	logger.GetLoggerFromCtx(ctx).Error(ctx, "failed processing a message: %w")
	return nil
}

func (k *KafkaReceiver[_]) sendToDLQ(givenMessage kafka.Message) {
	// TODO: DLQ with max retries
	panic("implement me")
}
