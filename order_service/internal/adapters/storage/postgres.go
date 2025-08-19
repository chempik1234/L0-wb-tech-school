package storage

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"order_service/internal/custom_errors"
	"order_service/internal/models"
)

type OrdersStoragePostgres struct {
	pool *pgxpool.Pool
}

func NewOrdersStoragePostgres(pool *pgxpool.Pool) *OrdersStoragePostgres {
	return &OrdersStoragePostgres{
		pool: pool,
	}
}

func (o *OrdersStoragePostgres) GetOrderById(ctx context.Context, orderId string) (models.Order, error) {
	var items []models.OrderItem
	var order models.Order

	// to parallel 2 queries, we launch them separately and check if they produced any non-nil errors
	// buffer of 2 prevents deadlock
	errorsChan := make(chan error, 2)

	go func() {
		var err error
		order, err = o.getOrderByIdBase(ctx, orderId)
		if err != nil {
			errorsChan <- fmt.Errorf("error while trying to get order itself: %w", err)
		} else {
			errorsChan <- nil
		}
	}()

	go func() {
		var err error
		items, err = o.getOrderItemsById(ctx, orderId)
		if err != nil {
			errorsChan <- fmt.Errorf("error while trying to order items: %w", err)
		} else {
			errorsChan <- nil
		}
	}()

	for i := 0; i < 2; i++ {
		if err := <-errorsChan; err != nil {
			return models.Order{}, err
		}
	}

	order.Items = items

	return order, nil
}

// getOrderByIdBase makes a long SELECT query to retrieve models.Order fields including models.Delivery, models.Payment
//
// instead of making many (3) async queries to many (3) different tables we make 1 big query
//
// call getOrderItemsById to find the items
func (o *OrdersStoragePostgres) getOrderByIdBase(ctx context.Context, orderId string) (models.Order, error) {
	// build select query
	sql, args, err := squirrel.Select(
		// order fields
		"o.order_uid", "o.track_number", "o.entry", "o.locale", "o.internal_signature",
		"o.customer_id", "o.delivery_service", "o.shardkey", "o.sm_id", "o.date_created",
		"o.oof_shard", "o.created_at", "o.updated_at",
		// delivery fields
		"d.order_id", "d.name", "d.phone", "d.zip", "d.city", "d.address", "d.region", "d.email",
		// payment fields
		"p.order_id", "p.transaction", "p.request_id", "p.currency", "p.provider", "p.amount",
		"p.payment_dt", "p.bank", "p.delivery_cost", "p.goods_total", "p.custom_fee",
	).
		From("order_service.orders o").
		Join("order_service.deliveries d ON d.order_id = o.order_uid").
		Join("order_service.payments p ON p.order_id = o.order_uid").
		Join("order_service.order_items i ON i.order_id = o.order_uid").
		Where(squirrel.Eq{"order_uid": orderId}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return models.Order{}, fmt.Errorf("Couldn't build and SQL query: %v", err)
	}

	var order models.Order

	// perform select query
	err = o.pool.QueryRow(context.Background(), sql, args...).Scan(
		// order fields
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerId, &order.DeliveryService, &order.ShardKey, &order.SmId, &order.DateCreated,
		&order.OofShard, &order.CreatedAt, &order.UpdatedAt,
		// delivery fields
		&order.Delivery.OrderId, &order.Delivery.Name, &order.Delivery.Phone,
		&order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
		&order.Delivery.Region, &order.Delivery.Email,
		// payment fields
		&order.Payment.OrderId, &order.Payment.Transaction, &order.Payment.RequestId,
		&order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount,
		&order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, custom_errors.ErrOrderNotFound
		}

		return models.Order{}, fmt.Errorf("error while mapping query result fields: %w", err)
	}

	if order.OrderUID == "" {
		return models.Order{}, custom_errors.ErrOrderNotFound
	}

	return order, nil
}

// getOrderItemsById makes a query to create a slice of models.OrderItem
// which can be used when retrieving models.Order
func (o *OrdersStoragePostgres) getOrderItemsById(ctx context.Context, orderId string) ([]models.OrderItem, error) {
	sql, args, err := squirrel.Select(
		"order_id", "chrt_id", "track_number", "price", "rid",
		"name", "sale", "size", "total_price", "nm_id", "brand", "status",
	).
		From("order_items").
		Where(squirrel.Eq{"order_id": orderId}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("couldn't build items query: %v", err)
	}

	var rows pgx.Rows
	rows, err = o.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("couldn't query items: %v", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err = rows.Scan(
			&item.OrderId, &item.ChrtId, &item.TrackNumber, &item.Price, &item.RId,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmId,
			&item.Brand, &item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan item: %v", err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (o *OrdersStoragePostgres) SaveOrder(ctx context.Context, order models.Order) error {
	//TODO implement me
	panic("implement me")
}
