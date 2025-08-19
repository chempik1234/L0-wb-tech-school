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
