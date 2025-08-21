package storage

import (
	"context"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/go-faster/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"order_service/internal/custom_errors"
	"order_service/internal/models"
	"strings"
)

// OrdersStoragePostgres is the postgres implementation of ports.OrderStorage
type OrdersStoragePostgres struct {
	pool *pgxpool.Pool
}

// NewOrdersStoragePostgres creates a new *OrdersStoragePostgres with given DB pool
func NewOrdersStoragePostgres(pool *pgxpool.Pool) *OrdersStoragePostgres {
	return &OrdersStoragePostgres{
		pool: pool,
	}
}

// GetOrderByID is the implementation of GetOrderByID method of ports.OrderStorage
//
// It gathers all data about the order with given ID if any.
//
// Querying for order and its items is parallel
func (o *OrdersStoragePostgres) GetOrderByID(ctx context.Context, orderID string) (models.Order, error) {
	var items []models.OrderItem
	var order models.Order

	eg, _ := errgroup.WithContext(ctx)

	eg.Go(func() error {
		var err error
		order, err = o.getOrderByIDBase(ctx, orderID)
		if err != nil {
			return fmt.Errorf("error trying to get order itself: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		items, err = o.getOrderItemsByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("error trying to order items: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return models.Order{}, err
	}

	order.Items = items

	return order, nil
}

// getOrderByIDBase makes a long SELECT query to retrieve models.Order fields including models.Delivery, models.Payment
//
// instead of making many (3) async queries to many (3) different tables we make 1 big query
//
// call getOrderItemsByID to find the items
func (o *OrdersStoragePostgres) getOrderByIDBase(ctx context.Context, orderID string) (models.Order, error) {
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
		Where(squirrel.Eq{"order_uid": orderID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return models.Order{}, fmt.Errorf("couldn't build and SQL query: %v", err)
	}

	var order models.Order

	// perform select query
	err = o.pool.QueryRow(context.Background(), sql, args...).Scan(
		// order fields
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SmID, &order.DateCreated,
		&order.OofShard, &order.CreatedAt, &order.UpdatedAt,
		// delivery fields
		&order.Delivery.OrderID, &order.Delivery.Name, &order.Delivery.Phone,
		&order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
		&order.Delivery.Region, &order.Delivery.Email,
		// payment fields
		&order.Payment.OrderID, &order.Payment.Transaction, &order.Payment.RequestID,
		&order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount,
		&order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, customerrors.ErrOrderNotFound
		}

		return models.Order{}, fmt.Errorf("error mapping query result fields: %w", err)
	}

	if order.OrderUID == "" {
		return models.Order{}, customerrors.ErrOrderNotFound
	}

	return order, nil
}

// getLastOrdersBase makes a long SELECT query to retrieve models.Order list, SORT BY created_at ASC
//
// it includes Payment and Delivery fields
//
// call getOrderItemsByID to get items for each of these
func (o *OrdersStoragePostgres) getLastOrdersBase(ctx context.Context, limit int) ([]models.Order, error) {
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
		OrderBy("o.created_at DESC").
		Limit(uint64(limit)).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return []models.Order{}, fmt.Errorf("couldn't build and SQL query: %v", err)
	}

	// perform select query
	var rows pgx.Rows
	rows, err = o.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("couldn't query last orders: %v", err)
	}
	defer rows.Close()

	var orders = make([]models.Order, 0)
	for rows.Next() {
		var order models.Order
		err = o.pool.QueryRow(context.Background(), sql, args...).Scan(
			// order fields
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
			&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SmID, &order.DateCreated,
			&order.OofShard, &order.CreatedAt, &order.UpdatedAt,
			// delivery fields
			&order.Delivery.OrderID, &order.Delivery.Name, &order.Delivery.Phone,
			&order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
			&order.Delivery.Region, &order.Delivery.Email,
			// payment fields
			&order.Payment.OrderID, &order.Payment.Transaction, &order.Payment.RequestID,
			&order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount,
			&order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
			&order.Payment.GoodsTotal, &order.Payment.CustomFee,
		)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan order: %v", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// getOrderItemsByID makes a query to create a slice of models.OrderItem
// which can be used when retrieving models.Order
func (o *OrdersStoragePostgres) getOrderItemsByID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	sql, args, err := squirrel.Select(
		"order_id", "chrt_id", "track_number", "price", "rid",
		"name", "sale", "size", "total_price", "nm_id", "brand", "status",
	).
		From("order_items").
		Where(squirrel.Eq{"order_id": orderID}).
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

	var items = make([]models.OrderItem, 0)
	for rows.Next() {
		var item models.OrderItem
		err = rows.Scan(
			&item.OrderID, &item.ChrtID, &item.TrackNumber, &item.Price, &item.RID,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID,
			&item.Brand, &item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("couldn't scan item: %v", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetLastOrders is implementation of such method in ports.OrderStorage
//
// It gets a few orders (limited by limit param) with biggest “created_at“
//
// Meant to be used in caching on startup
func (o *OrdersStoragePostgres) GetLastOrders(ctx context.Context, limit int) ([]models.Order, error) {
	ordersList, err := o.getLastOrdersBase(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("couldn't get last orders: %v", err)
	}

	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)

	itemsList := make([][]models.OrderItem, len(ordersList))

	for i, order := range ordersList {
		eg.Go(func() error {
			var items []models.OrderItem
			var itemsErr error

			items, itemsErr = o.getOrderItemsByID(ctx, order.OrderUID)
			if itemsErr != nil {
				return fmt.Errorf("error finding items for order %s: %w", order.OrderUID, itemsErr)
			}
			itemsList[i] = items
			return nil
		})
	}

	if err = eg.Wait(); err != nil {
		return nil, fmt.Errorf("couldn't get last orders items: %w", err)
	}

	for i, items := range itemsList {
		ordersList[i].Items = items
	}

	return ordersList, nil
}

// SaveOrder is implementation of such method in ports.OrderStorage
//
// It saves the order and related entities in a "long" transaction
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
			order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerID,
			order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard,
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

func savePayment(ctx context.Context, transaction pgx.Tx, orderUID string, payment *models.Payment) error {
	sql, args, err := squirrel.
		Insert("order_service.payments").
		Columns(
			"order_id", "transaction", "request_id", "currency", "provider",
			"amount", "payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
		).
		Values(
			orderUID, payment.Transaction, payment.RequestID, payment.Currency,
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

func saveDelivery(ctx context.Context, transaction pgx.Tx, orderUID string, delivery *models.Delivery) error {
	sql, args, err := squirrel.
		Insert("order_service.deliveries").
		Columns(
			"order_id", "name", "phone", "zip", "city", "address", "region", "email",
		).
		Values(
			orderUID, delivery.Name, delivery.Phone, delivery.Zip, delivery.City, delivery.Address,
			delivery.Region, delivery.Email,
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

func saveItems(ctx context.Context, transaction pgx.Tx, orderUID string, items *[]models.OrderItem) error {
	if len(*items) == 0 {
		return nil
	}

	valuesList := make([]string, len(*items))
	for i, item := range *items {
		valuesList[i] = fmt.Sprintf("('%s',%d,'%s',%d,'%s','%s',%d,'%s',%d,%d,'%s',%d)",
			orderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name, item.Sale, item.Size,
			item.TotalPrice, item.NmID, item.Brand, item.Status)
	}
	values := strings.Join(valuesList, ",")

	sql := fmt.Sprintf(
		"INSERT INTO order_service.order_items(\"order_id\", \"chrt_id\", \"track_number\", \"price\", \"rid\", \"name\", \"sale\", \"size\", \"total_price\", \"nm_id\", \"brand\", \"status\") VALUES %s",
		values,
	)

	_, err := transaction.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("couldn't exec save items query: %w", err)
	}
	return err
}
