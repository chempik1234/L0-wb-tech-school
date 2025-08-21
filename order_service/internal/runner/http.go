package runner

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"order_service/pkg/logger"
	"time"
)

// RunHTTP calls ListenAndServe on given srv, logs the beginning and the shutdown if on failure
func RunHTTP(ctx context.Context, srv *http.Server) {
	logger.GetLoggerFromCtx(ctx).Info(ctx, fmt.Sprintf("listening at %s", srv.Addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to serve gateway", zap.Error(err))
	}
}

// ShutdownHTTP stops httpServer with a 10 seconds timeout, logs on error
func ShutdownHTTP(ctx context.Context, httpServer *http.Server) {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := httpServer.Shutdown(cancelCtx)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "failed to shutdown http server", zap.Error(err))
	}
}
