package models

import (
	"encoding/json"
	"time"
)

type Order struct {
	ID          int             `json:"id"`
	UserID      int             `json:"user_id"`
	Items       json.RawMessage `json:"items"`
	TotalAmount float64         `json:"total_amount"`
	Currency    string          `json:"currency"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
