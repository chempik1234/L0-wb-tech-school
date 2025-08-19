package runner

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"order_service/pkg/logger"
	"time"
)

func RunHTTP(ctx context.Context, srv *http.Server) {
	logger.GetLoggerFromCtx(ctx).Info(ctx, fmt.Sprintf("listening at %s", srv.Addr))
	if err := srv.ListenAndServe(); err != nil {
		logger.GetLoggerFromCtx(ctx).Error(ctx, "failed to serve gateway", zap.Error(err))
	}
}

func ShutdownHTTP(ctx context.Context, httpServer *http.Server) {
	cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := httpServer.Shutdown(cancelCtx)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Warn(ctx, "failed to shutdown http server", zap.Error(err))
	}
}
