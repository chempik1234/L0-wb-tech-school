package customerrors

import "errors"

// ErrOrderNotFound describes an error when the storage
// was successfully checked but no order with given data was found
var ErrOrderNotFound = errors.New("order not found")
