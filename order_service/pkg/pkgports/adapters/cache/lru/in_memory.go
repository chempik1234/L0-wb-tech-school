package lru

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"order_service/pkg/linkedlist"
	"order_service/pkg/logger"
	"sync"
)

// ErrUnexpectedLinkedListBehaviour describes an error when linked list works wrong way
var ErrUnexpectedLinkedListBehaviour = errors.New("unexpected linked list behaviour")

// CacheLRUInMemory saves up to N Values and LRU algorithm and in-memory map storage
//
// It uses given key and value types, e.g. string and models.Order
//
// It uses sync.RWMutex because there are going to be many read operations from the web
type CacheLRUInMemory[Key comparable, Value any] struct {
	data     map[Key]Value
	keysList linkedlist.LinkedList[Key]
	mu       sync.RWMutex
	cap      int
}

// There are 2 options:
// A) store key index in keysList (in a separate map or in the data field)
// B) don't store key index and look it up every time I GET an element (LRU moves it to the top)
//
// option A:
//   1. after every SET, when data is inserted into linked list, we have to loop through N values and update index
//   2. after every GET, when key is moved to top, we have to loop through N-1 values and update index
// option B: after every GET, when data is retrieved from the list, we have to loop through N values to find the index
//
// we choose option B

// NewCacheLRUInMemory creates a new CacheLRUInMemory with given capacity and key/value types
//
// Example: myCache := NewCacheLRUInMemory[string, myStruct](myCapacity)
func NewCacheLRUInMemory[Key comparable, Value any](cacheCapacity int) *CacheLRUInMemory[Key, Value] {
	return &CacheLRUInMemory[Key, Value]{
		data:     make(map[Key]Value),
		keysList: linkedlist.NewLinkedList[Key](),
		cap:      cacheCapacity,
	}
}

// GetCapacity returns read-only value of CacheLRUInMemory capacity
func (c *CacheLRUInMemory[Key, Value]) GetCapacity() int {
	return c.cap
}

// Get tries to get an item by key, logs on miss
//
// It also moves read item to the top (if able)
func (c *CacheLRUInMemory[Key, Value]) Get(ctx context.Context, key Key) (Value, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.data[key]

	if ok {
		index, err := c.keysList.GetIndex(key, func(a, b Key) bool { return a == b })
		if err != nil {
			if errors.Is(err, linkedlist.ErrEmptyList) {
				return *new(Value), false, fmt.Errorf("%w: key is stored in data, but keys linked list is empty",
					ErrUnexpectedLinkedListBehaviour)
			}
			// not going to occur
			return value, false, fmt.Errorf("error getting index of read key: %w", err)
		}
		if index != -1 {
			err = c.keysList.MoveToFirst(index)
			if err != nil {
				return *new(Value), false, fmt.Errorf("error while putting element to top: %w", err)
			}
		} else {
			return *new(Value), false, fmt.Errorf("%w: key \"%v\" is stored in data, but not in non-empty linked list",
				ErrUnexpectedLinkedListBehaviour, key)
		}
	} else {
		logger.GetOrCreateLoggerFromCtx(ctx).Debug(ctx, "in-memory LRU cache miss", zap.Any("key", key))
	}

	return value, ok, nil
}

// Set saves the value
//
// moves it to the top as the most frequently checked
func (c *CacheLRUInMemory[Key, Value]) Set(ctx context.Context, key Key, value Value) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	index, err := c.keysList.GetIndex(key, func(a, b Key) bool { return a == b })
	if err != nil && !errors.Is(err, linkedlist.ErrEmptyList) {
		return fmt.Errorf("error trying to get existing entry of set key: %w", err)
	}
	if index != -1 {
		err = c.keysList.RemoveAt(index)
		if err != nil {
			return fmt.Errorf("error trying to remove existing entry of set key: %w", err)
		}
	}

	err = c.keysList.Insert(key, 0)
	if err != nil {
		return fmt.Errorf("error inserting key in list: %w", err)
	}

	c.data[key] = value

	// remove value if we're out of space
	if c.keysList.Len() > c.cap {
		var keyToDelete Key
		keyToDelete, err = c.keysList.GetLast()

		if err != nil {
			if !errors.Is(err, linkedlist.ErrEmptyList) {
				return fmt.Errorf("error while getting last key index: %w", err)
			}
		} else {
			err = c.keysList.RemoveLast()
			if err != nil {
				return fmt.Errorf("error while removing last key index: %w", err)
			}

			logger.GetLoggerFromCtx(ctx).Debug(ctx, "cache overflow, erased a value",
				zap.Any("key", keyToDelete), zap.Int("length", c.keysList.Len()),
				zap.Int("capacity", c.GetCapacity()))

			delete(c.data, keyToDelete)
		}
	}

	return nil
}

func (c *CacheLRUInMemory[_, _]) GetKeysAmount() int {
	return c.keysList.Len()
}

// GetKeys returns them in order from the Most to the least used
func (c *CacheLRUInMemory[Key, _]) GetKeys() []Key {
	return c.keysList.GetAll()
}

func (c *CacheLRUInMemory[Key, _]) MostUsedKey() (Key, error) {
	return c.keysList.GetFirst()
}

func (c *CacheLRUInMemory[Key, _]) LeastUsedKey() (Key, error) {
	return c.keysList.GetLast()
}
