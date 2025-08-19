package validators

import (
	"fmt"
	"net/mail"
	"order_service/internal/models"
	"strings"
	"time"
)

// Vibecoded because there are way too many fields

func ValidateOrder(order models.Order) error {
	if err := validateOrderMain(order); err != nil {
		return err
	}
	if err := validateDelivery(order.Delivery); err != nil {
		return fmt.Errorf("delivery validation failed: %w", err)
	}
	if err := validatePayment(order.Payment); err != nil {
		return fmt.Errorf("payment validation failed: %w", err)
	}
	if err := validateItems(order.Items); err != nil {
		return fmt.Errorf("items validation failed: %w", err)
	}
	return nil
}

func validateOrderMain(order models.Order) error {
	if order.OrderUID == "" {
		return fmt.Errorf("order_uid is required")
	}
	if strings.TrimSpace(order.TrackNumber) == "" {
		return fmt.Errorf("track_number is required")
	}
	if strings.TrimSpace(order.Entry) == "" {
		return fmt.Errorf("entry is required")
	}
	if strings.TrimSpace(order.Locale) == "" {
		return fmt.Errorf("locale is required")
	}
	if strings.TrimSpace(order.CustomerId) == "" {
		return fmt.Errorf("customer_id is required")
	}
	if strings.TrimSpace(order.DeliveryService) == "" {
		return fmt.Errorf("delivery_service is required")
	}
	if strings.TrimSpace(order.ShardKey) == "" {
		return fmt.Errorf("shardkey is required")
	}
	if order.SmId < 0 {
		return fmt.Errorf("sm_id must be non-negative")
	}
	if order.DateCreated.IsZero() {
		return fmt.Errorf("date_created is required")
	}
	if order.DateCreated.After(time.Now()) {
		return fmt.Errorf("date_created cannot be in the future")
	}
	if strings.TrimSpace(order.OofShard) == "" {
		return fmt.Errorf("oof_shard is required")
	}
	return nil
}

func validateDelivery(delivery models.Delivery) error {
	if strings.TrimSpace(delivery.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(delivery.Phone) == "" {
		return fmt.Errorf("phone is required")
	}
	if strings.TrimSpace(delivery.Zip) == "" {
		return fmt.Errorf("zip is required")
	}
	if strings.TrimSpace(delivery.City) == "" {
		return fmt.Errorf("city is required")
	}
	if strings.TrimSpace(delivery.Address) == "" {
		return fmt.Errorf("address is required")
	}
	if strings.TrimSpace(delivery.Region) == "" {
		return fmt.Errorf("region is required")
	}
	if strings.TrimSpace(delivery.Email) == "" {
		return fmt.Errorf("email is required")
	}
	// Basic email format validation
	if !isValidEmail(delivery.Email) {
		return fmt.Errorf("email has invalid format")
	}
	return nil
}

func validatePayment(payment models.Payment) error {
	if strings.TrimSpace(payment.Transaction) == "" {
		return fmt.Errorf("transaction is required")
	}
	if strings.TrimSpace(payment.Currency) == "" {
		return fmt.Errorf("currency is required")
	}
	if strings.TrimSpace(payment.Provider) == "" {
		return fmt.Errorf("provider is required")
	}
	if payment.Amount < 0 {
		return fmt.Errorf("amount must be non-negative")
	}
	if payment.PaymentDt <= 0 {
		return fmt.Errorf("payment_dt must be positive")
	}
	if strings.TrimSpace(payment.Bank) == "" {
		return fmt.Errorf("bank is required")
	}
	if payment.DeliveryCost < 0 {
		return fmt.Errorf("delivery_cost must be non-negative")
	}
	if payment.GoodsTotal < 0 {
		return fmt.Errorf("goods_total must be non-negative")
	}
	if payment.CustomFee < 0 {
		return fmt.Errorf("custom_fee must be non-negative")
	}
	return nil
}

func validateItems(items []models.OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("at least one item is required")
	}

	for i, item := range items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}
	return nil
}

func validateItem(item models.OrderItem) error {
	if item.ChrtId <= 0 {
		return fmt.Errorf("chrt_id must be positive")
	}
	if strings.TrimSpace(item.TrackNumber) == "" {
		return fmt.Errorf("track_number is required")
	}
	if item.Price < 0 {
		return fmt.Errorf("price must be non-negative")
	}
	if strings.TrimSpace(item.RId) == "" {
		return fmt.Errorf("rid is required")
	}
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if item.Sale < 0 || item.Sale > 100 {
		return fmt.Errorf("sale must be between 0 and 100")
	}
	if strings.TrimSpace(item.Size) == "" {
		return fmt.Errorf("size is required")
	}
	if item.TotalPrice < 0 {
		return fmt.Errorf("total_price must be non-negative")
	}
	if item.NmId <= 0 {
		return fmt.Errorf("nm_id must be positive")
	}
	if strings.TrimSpace(item.Brand) == "" {
		return fmt.Errorf("brand is required")
	}
	if item.Status < 0 {
		return fmt.Errorf("status must be non-negative")
	}
	return nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
