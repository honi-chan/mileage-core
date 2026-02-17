package mileage

import "context"

// MileageRepository はマイレージ集約のリポジトリインターフェース
type MileageRepository interface {
	// GetAccount はユーザーのマイレージ口座を取得する
	GetAccount(ctx context.Context, userID string) (*MileageAccount, error)

	// CreateAccount はマイレージ口座を新規作成する
	CreateAccount(ctx context.Context, userID string) error

	// UpdateBalance は楽観ロック付きで残高を更新する
	// expectedVersion と一致しない場合は ErrOptimisticLock を返す
	UpdateBalance(ctx context.Context, userID string, newBalance int64, expectedVersion int64) error

	// InsertTransaction は取引レコードを挿入する
	// 冪等性キーの重複時は ErrAlreadyProcessed または ErrIdempotencyConflict を返す
	InsertTransaction(ctx context.Context, tx *MileageTransaction) error

	// FindTransactionByIdempotencyKey は冪等性キーで取引を検索する
	FindTransactionByIdempotencyKey(ctx context.Context, userID string, idempotencyKey string, txType TransactionType) (*MileageTransaction, error)

	// ListTransactions はカーソルベースで取引一覧を取得する
	ListTransactions(ctx context.Context, userID string, cursor string, limit int) ([]*MileageTransaction, error)
}

// TxManager はDBトランザクションを管理するインターフェース
type TxManager interface {
	// RunInTx はトランザクション内で処理を実行する
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
