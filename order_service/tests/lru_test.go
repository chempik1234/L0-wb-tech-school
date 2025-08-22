package tests

import (
	"context"
	"order_service/pkg/logger"
	"order_service/pkg/pkgports/adapters/cache/lru"
	"testing"
)

func TestNewCacheLRUInMemory(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](10)

	if cache == nil {
		t.Error("Expected cache to be created")
	} else if cache.GetCapacity() != 10 {
		t.Errorf("Expected capacity 10, got %d", cache.GetCapacity())
	}
}

func TestCacheGetExistingKey(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](2)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// set value
	err = cache.Set(ctx, "test", 42)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// check value
	value, found, err := cache.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected key to be found")
	}
	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}
}

func TestCacheGetNonExistingKey(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](2)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// Try to get some imaginary key
	value, found, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected key not to be found")
	}
	if value != 0 {
		t.Errorf("Expected zero value, got %d", value)
	}
}

func TestCacheSetWithinCapacity(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](3)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// Set N values, N < capacity
	testCases := []struct {
		key   string
		value int
	}{
		{"key1", 1},
		{"key2", 2},
		{"key3", 3},
	}

	for _, tc := range testCases {
		err = cache.Set(ctx, tc.key, tc.value)
		if err != nil {
			t.Fatalf("Set failed for key %s: %v", tc.key, err)
		}
	}

	// Check that all N values are saved
	for _, tc := range testCases {
		value, found, err := cache.Get(ctx, tc.key)
		if err != nil {
			t.Fatalf("Get failed for key %s: %v", tc.key, err)
		}
		if !found {
			t.Errorf("Key %s should be found", tc.key)
		}
		if value != tc.value {
			t.Errorf("Expected value %d for key %s, got %d", tc.value, tc.key, value)
		}
	}
}

func TestCacheSetExceedCapacityLRUEviction(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](2)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	var keysList []string

	// Capacity is 2, we save 2 values to check both saved
	err = cache.Set(ctx, "key1", 1)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = cache.Set(ctx, "key2", 2)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Now the state is: key2, key1

	// key1 is the "least used" but it must be saved
	_, _, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Now the state is: key1, key2
	keysList = cache.GetKeys()
	if keysList[0] != "key1" || keysList[1] != "key2" {
		t.Errorf("After read key1 again, keys to be listed as key1, key2, got %v", keysList)
	}

	// Push key3 so key2 is erased
	err = cache.Set(ctx, "key3", 3)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Now the state is: key3, key1
	keysList = cache.GetKeys()
	if keysList[0] != "key3" || keysList[1] != "key1" {
		t.Errorf("After saving key3, keys to be listed as key3, key1, got %v", keysList)
	}

	// ... check if it really is
	_, found, err := cache.Get(ctx, "key2")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Key2 should have been evicted due to LRU policy")
	}

	// Now the state is: key3, key1

	// key3, key1 must be present
	value, found, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || value != 1 {
		t.Error("Key1 should still be in cache")
	}

	value, found, err = cache.Get(ctx, "key3")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || value != 3 {
		t.Error("Key3 should be in cache")
	}
}

func TestCacheZeroCapacity(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](0)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// If capacity is zero, then everything evaporates
	err = cache.Set(ctx, "test", 42)
	if err != nil {
		t.Fatalf("Set should handle zero capacity gracefully, got error: %v", err)
	}

	// Check if it does
	value, found, err := cache.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Value should not be found in zero-capacity cache")
	}
	if value != 0 {
		t.Errorf("Expected zero value, got %d", value)
	}
}

func TestCacheDifferentTypes(t *testing.T) {
	// Use different key-value types
	type customStruct struct {
		Field1 string
		Field2 int
	}

	// Key: int, Value: string
	cache1 := lru.NewCacheLRUInMemory[int, string](2)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	err = cache1.Set(ctx, 123, "hello")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, found, err := cache1.Get(ctx, 123)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || value != "hello" {
		t.Error("Int key test failed")
	}

	// Key: string, Value: some struct
	cache2 := lru.NewCacheLRUInMemory[string, customStruct](2)
	testStruct := customStruct{Field1: "test", Field2: 42}

	err = cache2.Set(ctx, "structKey", testStruct)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	result, found, err := cache2.Get(ctx, "structKey")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || result.Field1 != "test" || result.Field2 != 42 {
		t.Error("Custom struct value test failed")
	}
}

func TestCacheUpdateExistingKey(t *testing.T) {
	// As we know, indices are stored in a linked list
	// We need to ensure that old key is deleted from it even when the new one is the same

	cache := lru.NewCacheLRUInMemory[string, int](4)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// Set value for the 1st time
	err = cache.Set(ctx, "key1", 1)
	if err != nil {
		t.Fatalf("First set failed: %v", err)
	}

	// Set some CAPACITY-2 (=2) values to make key1 the least used
	err = cache.Set(ctx, "key2", 2)
	if err != nil {
		t.Fatalf("Random set failed: %v", err)
	}
	err = cache.Set(ctx, "key3", 3)
	if err != nil {
		t.Fatalf("Random set failed: %v", err)
	}

	// Set value for the 2nd time
	err = cache.Set(ctx, "key1", 100)
	if err != nil {
		t.Fatalf("Second set failed: %v", err)
	}

	// Check if it changed
	value, found, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || value != 100 {
		t.Errorf("Expected updated value 100, got %d", value)
	}

	// 4 saves, but 3 actual values
	err = cache.Set(ctx, "key4", 1)
	if err != nil {
		t.Fatalf("Random set failed: %v", err)
	}

	value, found, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Final get failed: %v", err)
	}
	if !found {
		t.Errorf("Expected updated value 100, but it erased (ensure key is stored only once in linked list")
	}
	if value != 100 {
		t.Errorf("Expected updated value 100, got %d", value)
	}
}

func TestKeysOrder(t *testing.T) {
	cache := lru.NewCacheLRUInMemory[string, int](3)
	ctx := context.Background()

	var err error
	ctx, err = logger.New(ctx)
	if err != nil {
		t.Fatalf("Error creating logger for test: %v", err)
	}

	// Set values
	err = cache.Set(ctx, "key1", 1)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = cache.Set(ctx, "key2", 1)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = cache.Set(ctx, "key3", 1)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	keysOrder := cache.GetKeys()
	if len(keysOrder) != 3 {
		t.Errorf("GetKeys failed: expected len 3, got: %d", len(keysOrder))
	}

	if keysOrder[0] != "key3" || keysOrder[1] != "key2" || keysOrder[2] != "key1" {
		t.Errorf("GetKeys failed: incorrect order, expected key3, key2, key1, got: %v", keysOrder)
	}

	// check most used key
	key, err := cache.MostUsedKey()
	if err != nil {
		t.Errorf("MostUsedKey failed: %v", err)
	}
	if key != "key3" {
		t.Errorf("MostUsedKey failed: incorrect order, expected key3, got: %v", key)
	}

	// check least used key
	key, err = cache.LeastUsedKey()
	if err != nil {
		t.Errorf("LeastUsedKey failed: %v", err)
	}
	if key != "key1" {
		t.Errorf("LeastUsedKey failed: incorrect order, expected key1, got: %v", key)
	}
}
