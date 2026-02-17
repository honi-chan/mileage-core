package mileage

import "time"

// TransactionType は取引種別
type TransactionType string

const (
	TransactionTypeGrant  TransactionType = "grant"
	TransactionTypeRedeem TransactionType = "redeem"
)

// MileageTransaction はマイレージ取引
type MileageTransaction struct {
	ID             string
	UserID         string
	Amount         int64
	Type           TransactionType
	IdempotencyKey string
	Reason         string
	RequestID      string
	CreatedAt      time.Time
}
