package service

import (
	"context"
	"go.uber.org/zap"
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/internal/validators"
	"order_service/pkg/logger"
)

type ProcessOrderFunction func(context.Context, models.Order) error

type OrderReceiverService[MessageType any] struct {
	receiver             ports.OrderReceiver[MessageType]
	processOrderFunction ProcessOrderFunction

	done chan struct{}
}

func NewOrderReceiverService[MessageType any](receiver ports.OrderReceiver[MessageType], processOrderFunction ProcessOrderFunction) *OrderReceiverService[MessageType] {
	return &OrderReceiverService[MessageType]{receiver: receiver, processOrderFunction: processOrderFunction, done: make(chan struct{})}
}

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
				break
			}

			// step 3: process
			go func() {
				err = s.ProcessOrder(ctx, order)
				if err != nil {
					logger.GetLoggerFromCtx(ctx).Error(ctx, "error while processing order", zap.Error(err))
					err = s.receiver.OnFail(ctx, msg)
					if err != nil {
						logger.GetLoggerFromCtx(ctx).Error(ctx, "error while committing message failure", zap.Error(err))
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

func (s *OrderReceiverService[_]) ProcessOrder(ctx context.Context, order models.Order) error {
	return s.processOrderFunction(ctx, order)
}

func (s *OrderReceiverService[_]) StopReceivingOrders(ctx context.Context) {
	// please tell me if it's nice or bad
	s.done <- struct{}{}
}
