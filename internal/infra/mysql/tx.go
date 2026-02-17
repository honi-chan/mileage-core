package mysql

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// txKey はコンテキストキー
type txKey struct{}

// TxManager はDBトランザクションを管理する
type TxManager struct {
	db *sqlx.DB
}

// NewTxManager は新しいTxManagerを作成する
func NewTxManager(db *sqlx.DB) *TxManager {
	return &TxManager{db: db}
}

// RunInTx はトランザクション内で処理を実行する
func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	ctx = context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetTx はコンテキストからトランザクションを取得する
// トランザクション外の場合はnilを返す
func GetTx(ctx context.Context) *sqlx.Tx {
	tx, _ := ctx.Value(txKey{}).(*sqlx.Tx)
	return tx
}

// Querier はDB操作のインターフェース（DBまたはTx）
type Querier interface {
	sqlx.QueryerContext
	sqlx.ExecerContext
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// GetQuerier はコンテキストからトランザクションまたはDBを返す
func GetQuerier(ctx context.Context, db *sqlx.DB) Querier {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return db
}
