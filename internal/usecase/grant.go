package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"

	domain "github.com/honi-chan/mileage-core/internal/domain/mileage"
	"github.com/honi-chan/mileage-core/internal/infra/redis"
)

// GrantUsecase はマイル付与ユースケース
type GrantUsecase struct {
	repo   domain.MileageRepository
	txMgr  domain.TxManager
	cache  *redis.Cache
	logger *zap.Logger
}

// NewGrantUsecase は新しいGrantUsecaseを作成する
func NewGrantUsecase(repo domain.MileageRepository, txMgr domain.TxManager, cache *redis.Cache, logger *zap.Logger) *GrantUsecase {
	return &GrantUsecase{
		repo:   repo,
		txMgr:  txMgr,
		cache:  cache,
		logger: logger,
	}
}

// GrantInput はマイル付与入力
type GrantInput struct {
	UserID         string
	Amount         int64
	Reason         string
	IdempotencyKey string
	RequestID      string
}

// GrantOutput はマイル付与出力
type GrantOutput struct {
	TransactionID string
	Balance       int64
	Version       int64
}

const maxRetries = 3

// Execute はマイル付与を実行する（楽観ロック + 冪等性 + リトライ）
func (uc *GrantUsecase) Execute(ctx context.Context, input GrantInput) (*GrantOutput, error) {
	if input.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if input.IdempotencyKey == "" {
		return nil, errors.New("idempotency key is required")
	}

	// 冪等性チェック: 同一キーの取引が存在するか
	existing, err := uc.repo.FindTransactionByIdempotencyKey(ctx, input.UserID, input.IdempotencyKey, domain.TransactionTypeGrant)
	if err != nil {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}
	if existing != nil {
		// 同じ内容なら同じ結果を返す
		if existing.Amount == input.Amount {
			account, err := uc.repo.GetAccount(ctx, input.UserID)
			if err != nil {
				return nil, fmt.Errorf("get account for idempotent response: %w", err)
			}
			return &GrantOutput{
				TransactionID: existing.ID,
				Balance:       account.Balance,
				Version:       account.Version,
			}, nil
		}
		// 異なる内容なら競合エラー
		return nil, domain.ErrIdempotencyConflict
	}

	// 楽観ロック付きリトライ
	var output *GrantOutput
	for attempt := 0; attempt < maxRetries; attempt++ {
		output, err = uc.executeInTx(ctx, input)
		if err == nil {
			break
		}
		if !errors.Is(err, domain.ErrOptimisticLock) {
			return nil, err
		}
		uc.logger.Warn("optimistic lock conflict, retrying",
			zap.Int("attempt", attempt+1),
			zap.String("user_id", input.UserID),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("grant failed after %d retries: %w", maxRetries, err)
	}

	// キャッシュ無効化
	if invalidateErr := uc.cache.InvalidateBalance(ctx, input.UserID); invalidateErr != nil {
		uc.logger.Warn("failed to invalidate cache", zap.Error(invalidateErr))
	}

	uc.logger.Info("mileage granted",
		zap.String("user_id", input.UserID),
		zap.Int64("amount", input.Amount),
		zap.String("transaction_id", output.TransactionID),
	)

	return output, nil
}

func (uc *GrantUsecase) executeInTx(ctx context.Context, input GrantInput) (*GrantOutput, error) {
	var output *GrantOutput

	err := uc.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. 口座取得
		account, err := uc.repo.GetAccount(txCtx, input.UserID)
		if err != nil {
			return fmt.Errorf("get account: %w", err)
		}
		if account == nil {
			return errors.New("account not found")
		}

		// 2. ドメインロジック（付与）
		expectedVersion := account.Version
		if err := account.Grant(input.Amount); err != nil {
			return err
		}

		// 3. 取引記録
		txID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
		tx := &domain.MileageTransaction{
			ID:             txID,
			UserID:         input.UserID,
			Amount:         input.Amount,
			Type:           domain.TransactionTypeGrant,
			IdempotencyKey: input.IdempotencyKey,
			Reason:         input.Reason,
			RequestID:      input.RequestID,
		}
		if err := uc.repo.InsertTransaction(txCtx, tx); err != nil {
			if errors.Is(err, domain.ErrAlreadyProcessed) {
				return err
			}
			return fmt.Errorf("insert transaction: %w", err)
		}

		// 4. 残高更新（楽観ロック）
		if err := uc.repo.UpdateBalance(txCtx, input.UserID, account.Balance, expectedVersion); err != nil {
			return err
		}

		output = &GrantOutput{
			TransactionID: txID,
			Balance:       account.Balance,
			Version:       account.Version,
		}
		return nil
	})

	return output, err
}
