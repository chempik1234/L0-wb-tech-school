package storage

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"order_service/internal/custom_errors"
	"order_service/internal/models"
	"strings"
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
			errorsChan <- fmt.Errorf("error trying to get order itself: %w", err)
		} else {
			errorsChan <- nil
		}
	}()

	go func() {
		var err error
		items, err = o.getOrderItemsById(ctx, orderId)
		if err != nil {
			errorsChan <- fmt.Errorf("error trying to order items: %w", err)
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

		return models.Order{}, fmt.Errorf("error mapping query result fields: %w", err)
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
	transaction, err := o.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("couldn't start transaction: %v", err)
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf("error saving order transaction, rolling back: %w", err)
			rollbackErr := transaction.Rollback(ctx)
			if rollbackErr != nil {
				err = fmt.Errorf("error rolling back transaction: %w. caused after this error: %w",
					rollbackErr, err)
			}
		}
		err = transaction.Commit(ctx)
	}()

	err = saveOrder(ctx, transaction, &order)
	if err != nil {
		return fmt.Errorf("couldn't save order: %w", err)
	}

	err = savePayment(ctx, transaction, order.OrderUID, &order.Payment)
	if err != nil {
		return fmt.Errorf("error saving payment: %w", err)
	}

	err = saveDelivery(ctx, transaction, order.OrderUID, &order.Delivery)
	if err != nil {
		return fmt.Errorf("error saving delivery: %w", err)
	}

	err = saveItems(ctx, transaction, order.OrderUID, &order.Items)
	if err != nil {
		return fmt.Errorf("error saving items: %w", err)
	}

	// check defer for more possible errors
	return err
}

func saveOrder(ctx context.Context, transaction pgx.Tx, order *models.Order) error {
	sql, args, err := squirrel.
		Insert("order_service.orders").
		Columns(
			"order_uid", "track_number", "entry", "locale", "internal_signature", "customer_id",
			"delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		).
		Values(
			order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerId,
			order.DeliveryService, order.ShardKey, order.SmId, order.DateCreated, order.OofShard,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("couldn't build an SQL query: %w", err)
	}

	var result pgconn.CommandTag
	result, err = transaction.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("couldn't exec save order query: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("couldn't save order query, rows affected: %d, expected: 1", result.RowsAffected())
	}
	return err
}

func savePayment(ctx context.Context, transaction pgx.Tx, orderUid string, payment *models.Payment) error {
	sql, args, err := squirrel.
		Insert("order_service.payments").
		Columns(
			"order_id", "transaction", "request_id", "currency", "provider",
			"amount", "payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
		).
		Values(
			orderUid, payment.Transaction, payment.RequestId, payment.Currency,
			payment.Provider, payment.Amount, payment.PaymentDt, payment.Bank,
			payment.DeliveryCost, payment.GoodsTotal, payment.CustomFee,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("couldn't build an SQL query: %w", err)
	}

	var result pgconn.CommandTag
	result, err = transaction.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("couldn't exec save payment query: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("couldn't save payment, rows affected: %d, expected: 1", result.RowsAffected())
	}
	return err
}

func saveDelivery(ctx context.Context, transaction pgx.Tx, orderUid string, delivery *models.Delivery) error {
	sql, args, err := squirrel.
		Insert("order_service.deliveries").
		Columns(
			"order_id", "name", "phone", "zip", "city", "address", "region", "email",
		).
		Values(
			orderUid, delivery.Name, delivery.Phone, delivery.Zip, delivery.City, delivery.Address,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("couldn't build an SQL query: %w", err)
	}

	var result pgconn.CommandTag
	result, err = transaction.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("couldn't exec save delivery query: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("couldn't save delivery, rows affected: %d, expected: 1", result.RowsAffected())
	}
	return err
}

func saveItems(ctx context.Context, transaction pgx.Tx, orderUid string, items *[]models.OrderItem) error {
	if len(*items) == 0 {
		return nil
	}

	valuesList := make([]string, len(*items))
	for i, item := range *items {
		valuesList[i] = fmt.Sprintf("(\"%s\",\"%d\",\"%s\",\"%d\",\"%s\",\"%s\",\"%d\",\"%s\",\"%d\",\"%d\",\"%s\",\"%d\")",
			orderUid, item.ChrtId, item.TrackNumber, item.Price, item.RId, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmId, item.Brand, item.Status)
	}
	values := strings.Join(valuesList, ",")

	sql := fmt.Sprintf(
		"INSERT INTO order_service.order_items(\"order_id\", \"chrt_id\", \"track_number\", \"price\", \"rid\", \"name\", \"sale\", \"size\", \"total_price\", \"nm_id\", \"brand\", \"status\") VALUES %s",
		values,
	)

	_, err := transaction.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("couldn't exec save delivery query: %w", err)
	}
	return err
}
