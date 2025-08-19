package ports

import (
	"context"
	"order_service/internal/models"
)

// OrderStorage port describes a persistent orders storage, e.g. postgres
type OrderStorage interface {
	GetOrderById(ctx context.Context, orderId string) (models.Order, error)
	SaveOrder(ctx context.Context, order models.Order) error
}

// OrderReceiver port describes a message queue consumer that gets orders for save, e.g. kafka
//
// it has an ability to be run and stopped (Run, process orders, GracefulStop)
type OrderReceiver[T any] interface {
	Consume(ctx context.Context) (models.Order, T, error)
	OnSuccess(ctx context.Context, givenMessage T) error
	OnFail(ctx context.Context, givenMessage T) error
}
