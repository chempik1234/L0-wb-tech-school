package service

import (
	"context"
	"order_service/internal/api"
)

type OrderService struct {
}

func NewOrderService() *OrderService {
	return new(OrderService)
}

func (s *OrderService) OrderIDGet(ctx context.Context, params api.OrderIDGetParams) (api.OrderIDGetRes, error) {
	response := api.OrderResponse{}
	return &response, nil
}
