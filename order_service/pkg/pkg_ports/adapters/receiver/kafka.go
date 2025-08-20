package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"order_service/pkg/logger"
	"order_service/pkg/pkg_ports"
	"time"
)

// KafkaMessage is the type that the service sees when receiving a message.
//
// stores the value to not deserialize it every retry
//
// stores the kafka.Message to commit success
//
// stores total tries, set to 0 if fresh message
type KafkaMessage[Value any] struct {
	Value      Value
	Message    kafka.Message
	RetryAfter time.Time
	TotalTries int
}

// NewFreshMessage creates a new *KafkaMessage[Value] as if it's just from kafka
//
// value is provided by the caller, message content ain't deserialized here
func NewFreshMessage[Value any](message kafka.Message, value Value) *KafkaMessage[Value] {
	return &KafkaMessage[Value]{
		Message:    message,
		Value:      value,
		TotalTries: 0,
	}
}

func NewRetriedMessage[Value any](message *KafkaMessage[Value], retryDelay time.Duration) *KafkaMessage[Value] {
	return &KafkaMessage[Value]{
		Value:      message.Value,
		Message:    message.Message,
		TotalTries: message.TotalTries + 1,
		RetryAfter: time.Now().Add(retryDelay),
	}
}

type KafkaReceiver[Value any] struct {
	reader       *kafka.Reader
	maxRetries   int
	retryChan    chan *KafkaMessage[Value] // this is wrong, check comment in Consume method
	fixedBackoff time.Duration
}

// To our disappointment, I didn't create generic interfaces for retry and backoff
// Skill issue

func NewKafkaReceiver[ValueType any](
	reader *kafka.Reader,
	maxRetries int, retriesCapacity int, fixedBackoff time.Duration,
) pkg_ports.Receiver[ValueType, *KafkaMessage[ValueType]] {
	return &KafkaReceiver[ValueType]{
		reader:       reader,
		maxRetries:   maxRetries,
		retryChan:    make(chan *KafkaMessage[ValueType], retriesCapacity),
		fixedBackoff: fixedBackoff,
	}
}

func (k *KafkaReceiver[Value]) Consume(ctx context.Context) (Value, *KafkaMessage[Value], error) {
	select {
	case failedMessage := <-k.retryChan:
		if time.Now().After(failedMessage.RetryAfter) {
			return failedMessage.Value, failedMessage, nil
		}

		// This is HELL wrong, pushing message to the end is bad
		// I could use a linked list instead
		// But I don't care yet
		k.retryChan <- failedMessage
	default:
		break // no retry messages
	}

	msg, err := k.reader.ReadMessage(ctx)
	if err != nil {
		return *new(Value), nil, fmt.Errorf("error while reading from kafka: %w", err)
	}
	var value Value
	err = json.Unmarshal(msg.Value, &value)
	if err != nil {
		return *new(Value), nil, fmt.Errorf("error while unmarshalling message: %w", err)
	}
	return value, NewFreshMessage[Value](msg, value), nil
}

func (k *KafkaReceiver[Value]) OnSuccess(ctx context.Context, givenMessage *KafkaMessage[Value]) error {
	return k.reader.CommitMessages(ctx, givenMessage.Message)
}

func (k *KafkaReceiver[Value]) OnFail(ctx context.Context, shouldRetry bool, givenMessage *KafkaMessage[Value]) error {
	if shouldRetry {
		k.sendToRetries(ctx, givenMessage)
	} else {
		k.sendToDLQ(ctx, givenMessage)
	}
	return nil
}

func (k *KafkaReceiver[Value]) sendToRetries(ctx context.Context, givenMessage *KafkaMessage[Value]) {
	totalTries := givenMessage.TotalTries + 1
	if totalTries > k.maxRetries {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "message sent to DLQ, max retries reached",
			zap.Int("total tries", totalTries))
		go k.sendToDLQ(ctx, givenMessage)
		return
	}

	newMessage := NewRetriedMessage[Value](givenMessage, k.fixedBackoff)
	select {
	case k.retryChan <- newMessage:
		logger.GetLoggerFromCtx(ctx).Info(ctx, "message sent to retry channel",
			zap.Int("total tries", newMessage.TotalTries), zap.Time("retry after", newMessage.RetryAfter))
	default:
		go k.sendToDLQ(ctx, givenMessage)
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "retry overflow! sending to DLQ",
			zap.Int("total tries", newMessage.TotalTries), zap.Time("retry after", newMessage.RetryAfter))
	}
}

func (k *KafkaReceiver[Value]) sendToDLQ(ctx context.Context, givenMessage *KafkaMessage[Value]) {
	// TODO: maybe real DLQ?
	logger.GetLoggerFromCtx(ctx).Info(ctx, "message sent to DLQ", zap.Any("message", givenMessage.Message))
}
