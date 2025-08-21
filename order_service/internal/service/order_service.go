package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

func (s *OrderService) GetLastOrders(ctx context.Context, limit int) ([]models.Order, error) {
	result, err := s.storage.GetLastOrders(ctx, limit)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error getting last orders by limit",
			zap.Int("limit", limit), zap.Error(err))
		return []models.Order{}, fmt.Errorf("error getting last orders by limit: %w", err)
	}
	return result, nil
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

func (s *OrderService) CacheLastOrders(ctx context.Context, limit int) error {
	lastOrders, err := s.storage.GetLastOrders(ctx, limit)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error getting last orders to cache",
			zap.Int("limit", limit), zap.Error(err))
		return fmt.Errorf("error getting last orders to cache: %w", err)
	}

	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)

	for _, order := range lastOrders {
		eg.Go(func() error {
			return s.cache.Set(ctx, order.OrderUID, order)
		})
	}

	if err = eg.Wait(); err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "error caching last orders to cache",
			zap.Int("limit", limit), zap.Error(err))
		return fmt.Errorf("error caching last orders to cache: %w", err)
	}
	return nil
}
