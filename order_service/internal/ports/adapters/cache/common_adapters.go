package cache

import (
	"order_service/internal/models"
	"order_service/internal/ports"
	"order_service/pkg/pkgports/adapters/cache/lru"
)

// NewOrderCacheAdapterInMemoryLRU creates a new lru.CacheLRUInMemory
//
// Adapter for service: string as KeyType and models.Order as ValueType
func NewOrderCacheAdapterInMemoryLRU(capacity int) ports.OrderCache {
	return lru.NewCacheLRUInMemory[string, models.Order](capacity)
}
