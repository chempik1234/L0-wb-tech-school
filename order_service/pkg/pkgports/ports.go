package pkgports

import (
	"context"
)

// Cache describes a cache that might be
// implemented with different storages (e.g. in-memory, redis)
// and mechanisms (e.g. N last saved)
type Cache[Key comparable, Value any] interface {
	// Set saves a value (invalidates first value)
	Set(ctx context.Context, key Key, value Value) error

	// Get returns value, ok, err (idempotent)
	Get(ctx context.Context, key Key) (Value, bool, error)

	// GetKeys returns a slice of all saved keys
	GetKeys() []Key

	// GetKeysAmount returns the amount of saved keys
	GetKeysAmount() int
}

// Receiver port describes a message queue consumer that gets orders for save, e.g. kafka
//
// values are read with Consume method and must be commited with either OnSuccess or OnFail
//
// values are unmarshalled into generic ValueType
//
// incoming messages that are passed into commit methods are MessageType (e.g. kafka.Message)
type Receiver[ValueType, MessageType any] interface {
	Consume(ctx context.Context) (ValueType, MessageType, error)
	// OnSuccess must be called on every successful message processing
	OnSuccess(ctx context.Context, givenMessage MessageType) error
	// OnFail must be called on every unsuccessful message processing
	OnFail(ctx context.Context, shouldRetry bool, givenMessage MessageType) error
}
