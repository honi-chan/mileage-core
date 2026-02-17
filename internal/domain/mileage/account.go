package mileage

import (
	"errors"
	"time"
)

// ドメインエラー
var (
	ErrInsufficientBalance = errors.New("balance is not enough")
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrOptimisticLock      = errors.New("optimistic lock conflict")
	ErrIdempotencyConflict = errors.New("idempotency key conflict with different content")
	ErrAlreadyProcessed    = errors.New("already processed with same idempotency key")
)

// MileageAccount はマイレージ口座の集約ルート
type MileageAccount struct {
	UserID    string
	Balance   int64
	Version   int64
	UpdatedAt time.Time
}

// Grant はマイルを付与する（不変条件チェック付き）
func (a *MileageAccount) Grant(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.Balance += amount
	a.Version++
	return nil
}

// Redeem はマイルを消費する（残高不足チェック付き）
func (a *MileageAccount) Redeem(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.Balance < amount {
		return ErrInsufficientBalance
	}
	a.Balance -= amount
	a.Version++
	return nil
}

// CanRedeem は消費可能かドライランで確認する
func (a *MileageAccount) CanRedeem(amount int64) bool {
	return amount > 0 && a.Balance >= amount
}
