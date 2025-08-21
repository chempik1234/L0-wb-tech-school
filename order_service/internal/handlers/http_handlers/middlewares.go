package http_handlers

import (
	"github.com/ogen-go/ogen/middleware"
	"go.uber.org/zap"
	"order_service/pkg/logger"
)

func LoggingMiddleware() middleware.Middleware {
	return func(
		req middleware.Request,
		next func(req middleware.Request) (middleware.Response, error),
	) (middleware.Response, error) {
		var err error
		req.Context, err = logger.New(req.Context)
		if err != nil {
			ctx := req.Context
			logger.GetOrCreateLoggerFromCtx(ctx).Error(ctx, "error creating logger for request",
				zap.Error(err))
		}

		var resp middleware.Response
		resp, err = next(req)

		if err != nil {
			ctx := req.Context
			logger.GetOrCreateLoggerFromCtx(ctx).Error(ctx, "response with error",
				zap.Error(err))
		}

		return resp, err
	}
}
