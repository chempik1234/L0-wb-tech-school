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

type OrderReceiverService struct {
	receiver             ports.OrderReceiver
	processOrderFunction ProcessOrderFunction

	done chan struct{}
}

func NewOrderReceiverService(receiver ports.OrderReceiver, processOrderFunction ProcessOrderFunction) *OrderReceiverService {
	return &OrderReceiverService{receiver: receiver, processOrderFunction: processOrderFunction, done: make(chan struct{})}
}

func (s *OrderReceiverService) StartReceivingOrders(ctx context.Context) error {
out:
	for {
		select {
		case <-ctx.Done():
			break out
		case <-s.done:
			break out
		default:
			// step 1: try to consume
			order, err := s.receiver.Consume(ctx)
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
				}
			}()
		}
	}
	return nil
}

func (s *OrderReceiverService) ProcessOrder(ctx context.Context, order models.Order) error {
	return s.processOrderFunction(ctx, order)
}

func (s *OrderReceiverService) StopReceivingOrders(ctx context.Context) {
	// please tell me if it's nice or bad
	s.done <- struct{}{}
}
