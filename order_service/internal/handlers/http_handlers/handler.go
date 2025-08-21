package http_handlers

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"order_service/internal/api"
	"order_service/internal/custom_errors"
	"order_service/internal/service"
	"order_service/pkg/logger"
)

// OrderServiceHttpHandler is a wrapper around service.OrderService that implements generated openapi handler
type OrderServiceHttpHandler struct {
	service *service.OrderService
}

func NewOrderServiceHttpHandler(service *service.OrderService) *OrderServiceHttpHandler {
	return &OrderServiceHttpHandler{
		service: service,
	}
}

func (s *OrderServiceHttpHandler) OrderIDGet(ctx context.Context, params api.OrderIDGetParams) (api.OrderIDGetRes, error) {
	orderUid := params.ID

	// query service
	result, err := s.service.GetOrder(ctx, orderUid)

	if err != nil {
		if errors.Is(err, custom_errors.ErrOrderNotFound) {
			return &api.NotFoundErrorResponse{
				Message: err.Error(),
			}, nil
		}

		// unknown error
		return &api.ErrorResponse{
			Message: fmt.Errorf("couldn't get order: %w", err).Error(),
		}, nil
	}

	items := make([]api.OrderItem, len(result.Items))
	for i, item := range result.Items {
		items[i] = api.OrderItem{
			ChrtID:      int64(item.ChrtId),
			TrackNumber: item.TrackNumber,
			Price:       item.Price,
			Rid:         item.RId,
			Name:        item.Name,
			Sale:        item.Sale,
			Size:        item.Size,
			TotalPrice:  item.TotalPrice,
			NmID:        int64(item.NmId),
			Brand:       item.Brand,
			Status:      item.Status,
		}
	}

	response := api.OrderResponse{
		OrderUID:    result.OrderUID,
		TrackNumber: result.TrackNumber,
		Entry:       result.Entry,
		Delivery: api.Delivery{
			Name:    result.Delivery.Name,
			Phone:   result.Delivery.Phone,
			Zip:     result.Delivery.Zip,
			City:    result.Delivery.City,
			Address: result.Delivery.Address,
			Region:  result.Delivery.Region,
			Email:   result.Delivery.Email,
		},
		Payment: api.Payment{
			Transaction: result.Payment.Transaction,
			RequestID: api.OptString{
				Value: result.Payment.RequestId,
				Set:   true,
			},
			Currency:     result.Payment.Currency,
			Provider:     result.Payment.Provider,
			Amount:       result.Payment.Amount,
			PaymentDt:    result.Payment.PaymentDt,
			Bank:         result.Payment.Bank,
			DeliveryCost: result.Payment.DeliveryCost,
			GoodsTotal:   result.Payment.GoodsTotal,
			CustomFee:    result.Payment.CustomFee,
		},
		Items:  items,
		Locale: result.Locale,
		InternalSignature: api.OptString{
			Set:   true,
			Value: result.InternalSignature,
		},
		CustomerID:      result.CustomerId,
		DeliveryService: result.DeliveryService,
		Shardkey:        result.ShardKey,
		SmID:            result.SmId,
		DateCreated:     result.DateCreated,
		OofShard:        result.OofShard,
	}

	logger.GetOrCreateLoggerFromCtx(ctx).Info(ctx, "read order by id", zap.String("order_uid", orderUid))

	return &response, nil
}

func (s *OrderServiceHttpHandler) NewError(ctx context.Context, err error) *api.ErrorResponseStatusCode {
	// handle custom errors whose status codes we know
	if errors.Is(err, custom_errors.ErrOrderNotFound) {
		return &api.ErrorResponseStatusCode{
			StatusCode: 404,
			Response: api.ErrorResponse{
				Message: err.Error(),
			},
		}
	}

	return &api.ErrorResponseStatusCode{
		StatusCode: 500,
		Response: api.ErrorResponse{
			Message: err.Error(),
		},
	}
}
