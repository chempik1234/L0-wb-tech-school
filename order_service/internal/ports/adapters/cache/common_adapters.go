package cache

import (
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/pkg/pkg_ports/adapters/cache/lru"
)

func NewOrderCacheAdapterInMemoryLRU(capacity int) ports.OrderCache {
	return lru.NewCacheLRUInMemory[string, models.Order](capacity)
}
