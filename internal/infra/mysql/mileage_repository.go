package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	domain "github.com/honi-chan/mileage-core/internal/domain/mileage"
	"github.com/jmoiron/sqlx"
)

// MileageRepository はマイレージリポジトリのMySQL実装
type MileageRepository struct {
	db *sqlx.DB
}

// NewMileageRepository は新しいMileageRepositoryを作成する
func NewMileageRepository(db *sqlx.DB) *MileageRepository {
	return &MileageRepository{db: db}
}

type accountRow struct {
	UserID    string       `db:"user_id"`
	Balance   int64        `db:"balance"`
	Version   int64        `db:"version"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}

type transactionRow struct {
	ID             string       `db:"id"`
	UserID         string       `db:"user_id"`
	Amount         int64        `db:"amount"`
	Type           string       `db:"type"`
	IdempotencyKey string       `db:"idempotency_key"`
	Reason         string       `db:"reason"`
	RequestID      string       `db:"request_id"`
	CreatedAt      sql.NullTime `db:"created_at"`
}

func (r *transactionRow) toDomain() *domain.MileageTransaction {
	tx := &domain.MileageTransaction{
		ID:             r.ID,
		UserID:         r.UserID,
		Amount:         r.Amount,
		Type:           domain.TransactionType(r.Type),
		IdempotencyKey: r.IdempotencyKey,
		Reason:         r.Reason,
		RequestID:      r.RequestID,
	}
	if r.CreatedAt.Valid {
		tx.CreatedAt = r.CreatedAt.Time
	}
	return tx
}

func (repo *MileageRepository) GetAccount(ctx context.Context, userID string) (*domain.MileageAccount, error) {
	q := GetQuerier(ctx, repo.db)
	var row accountRow
	err := q.GetContext(ctx, &row, "SELECT user_id, balance, version, updated_at FROM mileage_accounts WHERE user_id = ?", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	account := &domain.MileageAccount{
		UserID:  row.UserID,
		Balance: row.Balance,
		Version: row.Version,
	}
	if row.UpdatedAt.Valid {
		account.UpdatedAt = row.UpdatedAt.Time
	}
	return account, nil
}

func (repo *MileageRepository) CreateAccount(ctx context.Context, userID string) error {
	q := GetQuerier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		"INSERT INTO mileage_accounts (user_id, balance, version) VALUES (?, 0, 0)",
		userID,
	)
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

// UpdateBalance は楽観ロック付きで残高を更新する
func (repo *MileageRepository) UpdateBalance(ctx context.Context, userID string, newBalance int64, expectedVersion int64) error {
	q := GetQuerier(ctx, repo.db)
	result, err := q.ExecContext(ctx,
		"UPDATE mileage_accounts SET balance = ?, version = version + 1 WHERE user_id = ? AND version = ?",
		newBalance, userID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrOptimisticLock
	}
	return nil
}

// InsertTransaction は取引レコードを挿入する
func (repo *MileageRepository) InsertTransaction(ctx context.Context, tx *domain.MileageTransaction) error {
	q := GetQuerier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		"INSERT INTO mileage_transactions (id, user_id, amount, type, idempotency_key, reason, request_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		tx.ID, tx.UserID, tx.Amount, string(tx.Type), tx.IdempotencyKey, tx.Reason, tx.RequestID,
	)
	if err != nil {
		// MySQL duplicate entry エラーの判定
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			if strings.Contains(mysqlErr.Message, "uk_transactions_idempotency") {
				return domain.ErrAlreadyProcessed
			}
		}
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

// FindTransactionByIdempotencyKey は冪等性キーで取引を検索する
func (repo *MileageRepository) FindTransactionByIdempotencyKey(ctx context.Context, userID string, idempotencyKey string, txType domain.TransactionType) (*domain.MileageTransaction, error) {
	q := GetQuerier(ctx, repo.db)
	var row transactionRow
	err := q.GetContext(ctx, &row,
		"SELECT id, user_id, amount, type, idempotency_key, reason, request_id, created_at FROM mileage_transactions WHERE user_id = ? AND idempotency_key = ? AND type = ?",
		userID, idempotencyKey, string(txType),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find transaction by idempotency key: %w", err)
	}
	return row.toDomain(), nil
}

// ListTransactions はカーソルベースで取引一覧を取得する
func (repo *MileageRepository) ListTransactions(ctx context.Context, userID string, cursor string, limit int) ([]*domain.MileageTransaction, error) {
	q := GetQuerier(ctx, repo.db)
	var rows []transactionRow

	if cursor == "" {
		err := q.SelectContext(ctx, &rows,
			"SELECT id, user_id, amount, type, idempotency_key, reason, request_id, created_at FROM mileage_transactions WHERE user_id = ? ORDER BY created_at DESC LIMIT ?",
			userID, limit+1,
		)
		if err != nil {
			return nil, fmt.Errorf("list transactions: %w", err)
		}
	} else {
		err := q.SelectContext(ctx, &rows,
			"SELECT id, user_id, amount, type, idempotency_key, reason, request_id, created_at FROM mileage_transactions WHERE user_id = ? AND id < ? ORDER BY created_at DESC LIMIT ?",
			userID, cursor, limit+1,
		)
		if err != nil {
			return nil, fmt.Errorf("list transactions with cursor: %w", err)
		}
	}

	result := make([]*domain.MileageTransaction, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.toDomain())
	}
	return result, nil
}
