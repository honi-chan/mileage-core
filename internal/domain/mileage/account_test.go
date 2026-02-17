package mileage_test

import (
	"testing"

	"github.com/honi-chan/mileage-core/internal/domain/mileage"
)

func TestMileageAccount_Grant(t *testing.T) {
	tests := []struct {
		name        string
		balance     int64
		amount      int64
		wantBalance int64
		wantErr     error
	}{
		{
			name:        "正常な付与",
			balance:     100,
			amount:      50,
			wantBalance: 150,
		},
		{
			name:        "残高0からの付与",
			balance:     0,
			amount:      100,
			wantBalance: 100,
		},
		{
			name:    "0マイルの付与はエラー",
			balance: 100,
			amount:  0,
			wantErr: mileage.ErrInvalidAmount,
		},
		{
			name:    "負の数の付与はエラー",
			balance: 100,
			amount:  -10,
			wantErr: mileage.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &mileage.MileageAccount{
				UserID:  "test-user",
				Balance: tt.balance,
				Version: 1,
			}

			err := account.Grant(tt.amount)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Grant() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Grant() unexpected error: %v", err)
			}

			if account.Balance != tt.wantBalance {
				t.Errorf("Balance = %d, want %d", account.Balance, tt.wantBalance)
			}

			if account.Version != 2 {
				t.Errorf("Version = %d, want 2", account.Version)
			}
		})
	}
}

func TestMileageAccount_Redeem(t *testing.T) {
	tests := []struct {
		name        string
		balance     int64
		amount      int64
		wantBalance int64
		wantErr     error
	}{
		{
			name:        "正常な消費",
			balance:     200,
			amount:      100,
			wantBalance: 100,
		},
		{
			name:        "全額消費",
			balance:     100,
			amount:      100,
			wantBalance: 0,
		},
		{
			name:    "残高不足",
			balance: 50,
			amount:  100,
			wantErr: mileage.ErrInsufficientBalance,
		},
		{
			name:    "0マイルの消費はエラー",
			balance: 100,
			amount:  0,
			wantErr: mileage.ErrInvalidAmount,
		},
		{
			name:    "負の数の消費はエラー",
			balance: 100,
			amount:  -10,
			wantErr: mileage.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &mileage.MileageAccount{
				UserID:  "test-user",
				Balance: tt.balance,
				Version: 1,
			}

			err := account.Redeem(tt.amount)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Redeem() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Redeem() unexpected error: %v", err)
			}

			if account.Balance != tt.wantBalance {
				t.Errorf("Balance = %d, want %d", account.Balance, tt.wantBalance)
			}

			if account.Version != 2 {
				t.Errorf("Version = %d, want 2", account.Version)
			}
		})
	}
}

func TestMileageAccount_CanRedeem(t *testing.T) {
	account := &mileage.MileageAccount{Balance: 100}

	if !account.CanRedeem(50) {
		t.Error("CanRedeem(50) should be true")
	}
	if !account.CanRedeem(100) {
		t.Error("CanRedeem(100) should be true")
	}
	if account.CanRedeem(101) {
		t.Error("CanRedeem(101) should be false")
	}
	if account.CanRedeem(0) {
		t.Error("CanRedeem(0) should be false")
	}
	if account.CanRedeem(-1) {
		t.Error("CanRedeem(-1) should be false")
	}
}
