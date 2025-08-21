package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/pkg/logger"
)

type OrderService struct {
	storage ports.OrderStorage
	cache   ports.OrderCache
}

func NewOrderService(storage ports.OrderStorage, cache ports.OrderCache) *OrderService {
	return &OrderService{
		storage: storage,
		cache:   cache,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, orderUid string) (models.Order, error) {
	// step 1. try to check cache first
	result, found, err := s.cache.Get(ctx, orderUid)
	if err != nil {
		return models.Order{}, fmt.Errorf("error checking orders cache: %w", err)
	}

	if !found {
		// step 2. call the storage if not found in cache
		result, err = s.storage.GetOrderById(ctx, orderUid)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "error retrieving order from storage",
				zap.String("key", orderUid), zap.Error(err))
			return models.Order{}, err
		}

		// step 3. cache the value
		go func() {
			cacheErr := s.cache.Set(ctx, result.OrderUID, result)
			if err != nil {
				logger.GetLoggerFromCtx(ctx).Error(ctx, "error caching order",
					zap.String("key", orderUid), zap.Error(cacheErr))
			}
		}()

	}

	return result, err
}

func (s *OrderService) SaveOrder(ctx context.Context, order models.Order) error {
	// step 1. try to save in storage
	err := s.storage.SaveOrder(ctx, order)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error saving order",
			zap.String("key", order.OrderUID), zap.Error(err))
		return err
	}

	// step 2. cache it for the future
	//   only if value was successfully saved
	go func() {
		cacheErr := s.cache.Set(ctx, order.OrderUID, order)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "error caching order",
				zap.String("key", order.OrderUID), zap.Error(cacheErr))
		}
	}()

	return nil
}
