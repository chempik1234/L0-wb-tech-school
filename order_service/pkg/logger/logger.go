package logger

import (
	"context"
	"go.uber.org/zap"
)

type key string

const (
	// KeyForLogger is used to store Logger in a context.Context
	KeyForLogger key = "logger"
	// KeyForRequestID is used to store some request ID in a context.Context, purely optional to use
	KeyForRequestID key = "request_id"
)

// Logger is a type that stores a pointer on zap.Logger
//
// Supposed to be stored in context.Context
type Logger struct {
	l *zap.Logger
}

// NewLogger creates a new Logger, might return an error because of zap
func NewLogger() (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	loggerStruct := &Logger{l: logger}

	return loggerStruct, nil
}

// New creates a new context.Context with a new logger placed in it
func New(ctx context.Context) (context.Context, error) {
	loggerStruct, err := NewLogger()
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, KeyForLogger, loggerStruct)
	return ctx, nil
}

// GetLoggerFromCtx gets Logger from given ctx if present, else panic
func GetLoggerFromCtx(ctx context.Context) *Logger {
	return ctx.Value(KeyForLogger).(*Logger)
}

// TryAppendRequestIDFromContext appends a field with ID of current request if it's in given context
// (check KeyForRequestID)
func TryAppendRequestIDFromContext(ctx context.Context, fields []zap.Field) []zap.Field {
	if ctx.Value(KeyForRequestID) != nil {
		fields = append(fields, zap.String(string(KeyForRequestID), ctx.Value(KeyForRequestID).(string)))
	}
	return fields
}

// GetOrCreateLoggerFromCtx is a safe version on GetLoggerFromCtx that creates a new logger if no logger is in ctx
func GetOrCreateLoggerFromCtx(ctx context.Context) *Logger {
	logger := GetLoggerFromCtx(ctx)
	if logger == nil {
		logger, _ = NewLogger()
	}
	return logger
}

// Debug makes a debug level message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = TryAppendRequestIDFromContext(ctx, fields)
	l.l.Debug(msg, fields...)
}

// Info makes an info level message
func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = TryAppendRequestIDFromContext(ctx, fields)
	l.l.Info(msg, fields...)
}

// Warn makes a warn level message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = TryAppendRequestIDFromContext(ctx, fields)
	l.l.Warn(msg, fields...)
}

// Error makes an error level message
func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = TryAppendRequestIDFromContext(ctx, fields)
	l.l.Error(msg, fields...)
}

// Fatal makes a fatal level message
func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	fields = TryAppendRequestIDFromContext(ctx, fields)
	l.l.Fatal(msg, fields...)
}
