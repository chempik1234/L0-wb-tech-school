package service

import (
	"context"
	"go.uber.org/zap"
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/internal/validators"
	"order_service/pkg/logger"
)

// ProcessOrderFunction is the type of function that can be called on each received order
type ProcessOrderFunction func(context.Context, models.Order) error

// OrderReceiverService is a service that reads the orders continuously, validates and processes them
//
// It supports different implementations, so MessageType is generic
type OrderReceiverService[MessageType any] struct {
	receiver             ports.OrderReceiver[MessageType]
	processOrderFunction ProcessOrderFunction

	done chan struct{}
}

// NewOrderReceiverService creates a new receiver service with given receiver repository and process function
func NewOrderReceiverService[MessageType any](receiver ports.OrderReceiver[MessageType], processOrderFunction ProcessOrderFunction) *OrderReceiverService[MessageType] {
	return &OrderReceiverService[MessageType]{receiver: receiver, processOrderFunction: processOrderFunction, done: make(chan struct{})}
}

// StartReceivingOrders is the main loop function that is meant to be run in background
func (s *OrderReceiverService[_]) StartReceivingOrders(ctx context.Context) error {
out:
	for {
		select {
		case <-ctx.Done():
			break out
		case <-s.done:
			break out
		default:
			// step 1: try to consume
			order, msg, err := s.receiver.Consume(ctx)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "error while receiving orders",
					zap.Error(err))
				break
			}

			// step 2: validate
			err = validators.ValidateOrder(order)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Warn(ctx, "invalid order", zap.Error(err))

				// message is incorrect, no retries
				err = s.receiver.OnFail(ctx, false, msg)
				if err != nil {
					logger.GetLoggerFromCtx(ctx).Error(ctx, "error while committing invalid message failure", zap.Error(err))
				}
				break
			}

			// step 3: process
			go func() {
				err = s.ProcessOrder(ctx, order)
				if err != nil {
					logger.GetLoggerFromCtx(ctx).Error(ctx, "error while processing order", zap.Error(err))

					// send message to retry because of unknown DB errors
					err = s.receiver.OnFail(ctx, true, msg)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "error while committing valid message failure", zap.Error(err))
					}
				} else {
					err = s.receiver.OnSuccess(ctx, msg)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "error while committing successful message", zap.Error(err))
					}
				}
			}()
		}
	}
	return nil
}

// ProcessOrder is called on every valid order, calls processOrderFunction
// provided in NewOrderReceiverService
func (s *OrderReceiverService[_]) ProcessOrder(ctx context.Context, order models.Order) error {
	return s.processOrderFunction(ctx, order)
}

// StopReceivingOrders sends a signal to stop looping in the StartReceivingOrders
func (s *OrderReceiverService[_]) StopReceivingOrders(ctx context.Context) {
	// please tell me if it's nice or bad
	s.done <- struct{}{}
}
