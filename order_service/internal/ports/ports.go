package ports

import (
	"context"
	"order_service/internal/models"
	"order_service/pkg/pkgports"
)

// OrderStorage port describes a persistent orders storage, e.g. postgres
type OrderStorage interface {
	GetOrderByID(ctx context.Context, orderID string) (models.Order, error)
	GetLastOrders(ctx context.Context, limit int) ([]models.Order, error)
	SaveOrder(ctx context.Context, order models.Order) error
}

// OrderReceiver port describes a message queue consumer that gets orders for save, e.g. kafka
//
// values are read with Consume method and must be commited with either OnSuccess or OnFail
type OrderReceiver[MessageType any] pkgports.Receiver[models.Order, MessageType]

// OrderCache describes a cache that might be
// implemented with different storages (e.g. in-memory, redis)
// and mechanisms (e.g. N last saved)
type OrderCache pkgports.Cache[string, models.Order]
