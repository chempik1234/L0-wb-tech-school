package models

import (
	"time"
)

type Order struct {
	OrderUID          string
	TrackNumber       string
	Entry             string
	Locale            string
	InternalSignature string
	CustomerId        string
	DeliveryService   string
	ShardKey          string
	SmId              int
	DateCreated       time.Time
	OofShard          string
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Delivery Delivery
	Payment  Payment
	Items    []OrderItem
}

type Delivery struct {
	OrderId string
	Name    string
	Phone   string
	Zip     string
	City    string
	Address string
	Region  string
	Email   string
}

type Payment struct {
	OrderId      string
	Transaction  string
	RequestId    string
	Currency     string
	Provider     string
	Amount       int
	PaymentDt    int64
	Bank         string
	DeliveryCost int
	GoodsTotal   int
	CustomFee    int
}

type OrderItem struct {
	OrderId     string
	ChrtId      int
	TrackNumber string
	Price       int
	RId         string
	Name        string
	Sale        int
	Size        string
	TotalPrice  int
	NmId        int
	Brand       string
	Status      int
}
