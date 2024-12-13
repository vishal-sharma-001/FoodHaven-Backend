package models

import (
	"time"
)

type Order struct {
	OrderID     int         `json:"order_id"`
	UserID      int         `json:"user_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Currency    string      `json:"currency"`
	Status      string      `json:"status"`
	Txnid       string      `json:"txnid"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Price        float64    `json:"price"`
	CloudImageID string `json:"cloudimageid"`
	Quantity     int    `json:"quantity"`
	RestaurantID int    `json:"restrauntId"`
}

type PaymentRequest struct {
	Items   []OrderItem `json:"items"`
	Amount  int `json:"amount"`
}
