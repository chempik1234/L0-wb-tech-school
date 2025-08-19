package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"order_service/internal/adapters/storage"
	"order_service/internal/api"
	"order_service/internal/config"
	"order_service/internal/runner"
	"order_service/internal/service"
	"order_service/pkg/kafka"
	"order_service/pkg/logger"
	"order_service/pkg/postgres"
	"os"
	"os/signal"
	"sync"
)

func main() {
	ctx := context.Background()

	// use OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	// put a new zap logger into context
	// from now on, all packages PURELY HOPE that the logger is there (otherwise the service explodes)
	ctx, _ = logger.New(ctx)

	// read config from whatever (.env)
	cfg, err := config.TryRead()
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to load config", zap.Error(err))
	}

	pgCfg := cfg.Postgres
	kafkaCfg := cfg.Kafka
	serviceCfg := cfg.OrderService

	//region connections

	pool, err := postgres.New(ctx, pgCfg)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to connect to postgres", zap.Error(err))
	}
	logger.GetLoggerFromCtx(ctx).Info(ctx, "connected to postgres")

	err = kafka.CreateTopicIfNotExists(kafkaCfg, serviceCfg.KafkaTopic, 1, 1)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to create topic kafka", zap.Error(err))
	}
	kafkaConsumer := kafka.NewReader(ctx,
		kafkaCfg, serviceCfg.KafkaTopic, serviceCfg.KafkaGroupID)
	//endregion

	//region service
	storageAdapter := storage.NewOrdersStoragePostgres(pool)
	orderService := service.NewOrderService(storageAdapter)
	orderServiceHandler := service.NewOrderServiceAPIWrapper(orderService)
	//endregion

	// create handler aka mux from ogen-generated function
	// using the service
	apiHandler, err := api.NewServer(orderServiceHandler)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "failed to create http server", zap.Error(err))
	}

	// create and let run http server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", serviceCfg.HTTPPort),
		Handler: apiHandler,
	}
	go runner.RunHTTP(ctx, httpServer)

	<-ctx.Done()

	var shutdownWg sync.WaitGroup
	shutdownWg.Add(3)

	// shutdowns don't include wg itself, so I wrap them in unnamed goroutines
	go func() {
		defer shutdownWg.Done()
		runner.ShutdownHTTP(ctx, httpServer)
		logger.GetLoggerFromCtx(ctx).Info(ctx, "server stopped")
	}()
	go func() {
		defer shutdownWg.Done()
		pool.Close()
		logger.GetLoggerFromCtx(ctx).Info(ctx, "postgres pool stopped")
	}()
	go func() {
		defer shutdownWg.Done()
		err = kafkaConsumer.Close()
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Error(ctx, "error while closing kafka consumer", zap.Error(err))
		}
		logger.GetLoggerFromCtx(ctx).Info(ctx, "kafka consumer stopped")
	}()

	shutdownWg.Wait()
}
