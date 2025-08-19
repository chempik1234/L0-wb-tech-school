package runner

import (
	"context"
	"go.uber.org/zap"
	"order_service/internal/service"
	"order_service/pkg/logger"
	"time"
)

func RunOrderReceiver[T any](ctx context.Context, receiver *service.OrderReceiverService[T]) {
	logger.GetLoggerFromCtx(ctx).Info(ctx, "starting receiving orders")
	if err := receiver.StartReceivingOrders(ctx); err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to receive orders", zap.Error(err))
	}
}

func ShutdownOrderReceiver[T any](ctx context.Context, receiver *service.OrderReceiverService[T]) {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receiver.StopReceivingOrders(cancelCtx)
}
