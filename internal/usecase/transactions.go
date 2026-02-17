package usecase

import (
	"context"
	"fmt"

	domain "github.com/honi-chan/mileage-core/internal/domain/mileage"
)

// TransactionsUsecase は取引履歴ユースケース
type TransactionsUsecase struct {
	repo domain.MileageRepository
}

// NewTransactionsUsecase は新しいTransactionsUsecaseを作成する
func NewTransactionsUsecase(repo domain.MileageRepository) *TransactionsUsecase {
	return &TransactionsUsecase{repo: repo}
}

// TransactionsInput は取引履歴入力
type TransactionsInput struct {
	UserID string
	Cursor string
	Limit  int
}

// TransactionItem は取引履歴の1件
type TransactionItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

// TransactionsOutput は取引履歴出力
type TransactionsOutput struct {
	Items      []TransactionItem `json:"items"`
	NextCursor string            `json:"next_cursor"`
}

// Execute は取引履歴を取得する（カーソルベースページネーション）
func (uc *TransactionsUsecase) Execute(ctx context.Context, input TransactionsInput) (*TransactionsOutput, error) {
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 50
	}

	txs, err := uc.repo.ListTransactions(ctx, input.UserID, input.Cursor, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	output := &TransactionsOutput{
		Items: make([]TransactionItem, 0),
	}

	for i, tx := range txs {
		if i >= input.Limit {
			// limit+1件目はnext_cursorとして使用
			output.NextCursor = tx.ID
			break
		}
		output.Items = append(output.Items, TransactionItem{
			ID:        tx.ID,
			Type:      string(tx.Type),
			Amount:    tx.Amount,
			Reason:    tx.Reason,
			CreatedAt: tx.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return output, nil
}
