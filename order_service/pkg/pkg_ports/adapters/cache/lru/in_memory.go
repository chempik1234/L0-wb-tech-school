package lru

import (
	"context"
	"fmt"
	"order_service/pkg/linked_list"
	"sync"
)

type lruKey[Value any] struct {
	Value Value
	Index int
}

// CacheLRUInMemory saves up to N Values and LRU algorithm and in-memory map storage
//
// It uses given key and value types, e.g. string and models.Order
//
// It uses sync.RWMutex because there are going to be many read operations from the web
type CacheLRUInMemory[Key comparable, Value any] struct {
	data     map[Key]lruKey[Value]
	keysList linked_list.LinkedList[Key]
	mu       sync.RWMutex
	cap      int
}

// There are 2 options:
// A) store key index in keysList (in a separate map or in the data field)
// B) don't store key index and look it up every time I GET an element (LRU moves it to the top)
//
// option A: after every SET, when data is inserted into linked list, we have to loop through N values and update index
// option B: after every GET, when data is retrieved from the list, we have to loop through N values to find the index
//
// GET happens more often, so we choose option A

func NewCacheLRUInMemory[Key comparable, Value any](cacheCapacity int) *CacheLRUInMemory[Key, Value] {
	return &CacheLRUInMemory[Key, Value]{
		data:     make(map[Key]lruKey[Value]),
		keysList: linked_list.NewLinkedList[Key](),
		cap:      cacheCapacity,
	}
}

func (c *CacheLRUInMemory[Key, Value]) Get(ctx context.Context, key Key) (Value, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.data[key]

	if ok {
		err := c.keysList.MoveToFirst(value.Index)
		if err != nil {
			return *new(Value), false, fmt.Errorf("error while putting element to top: %w", err)
		}
	}

	return value.Value, ok, nil
}

// Set saves the value
//
// moves it to the top as the most frequently checked
func (c *CacheLRUInMemory[Key, Value]) Set(ctx context.Context, key Key, value Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// remove value if we're out of space
	if c.keysList.Len() > c.cap {
		keyToDelete, err := c.keysList.GetLast()
		if err != nil {
			return fmt.Errorf("error while getting last key index: %w", err)
		}

		delete(c.data, keyToDelete)
		c.keysList.RemoveLast()
	}

	err := c.keysList.Insert(key, 0)
	if err != nil {
		return fmt.Errorf("error inserting key in list: %w", err)
	}

	c.data[key] = lruKey[Value]{Index: 0, Value: value}
	return nil
}
