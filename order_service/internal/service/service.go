package service

import (
	"context"
	"order_service/internal/models"
	"order_service/internal/ports"
)

type OrderService struct {
	storage ports.OrderStorage
}

func NewOrderService(storage ports.OrderStorage) *OrderService {
	return &OrderService{
		storage: storage,
	}
}

func (s *OrderService) OrderIDGet(ctx context.Context, orderUid string) (models.Order, error) {
	// call the storage if not found in cache
	result, err := s.storage.GetOrderById(ctx, orderUid)

	return result, err
}
