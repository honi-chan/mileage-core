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

// RedeemUsecase はマイル消費ユースケース
type RedeemUsecase struct {
	repo   domain.MileageRepository
	txMgr  domain.TxManager
	cache  *redis.Cache
	logger *zap.Logger
}

// NewRedeemUsecase は新しいRedeemUsecaseを作成する
func NewRedeemUsecase(repo domain.MileageRepository, txMgr domain.TxManager, cache *redis.Cache, logger *zap.Logger) *RedeemUsecase {
	return &RedeemUsecase{
		repo:   repo,
		txMgr:  txMgr,
		cache:  cache,
		logger: logger,
	}
}

// RedeemInput はマイル消費入力
type RedeemInput struct {
	UserID         string
	Amount         int64
	Reason         string
	IdempotencyKey string
	RequestID      string
}

// RedeemOutput はマイル消費出力
type RedeemOutput struct {
	TransactionID string
	Balance       int64
	Version       int64
}

// Execute はマイル消費を実行する（楽観ロック + 冪等性 + リトライ）
func (uc *RedeemUsecase) Execute(ctx context.Context, input RedeemInput) (*RedeemOutput, error) {
	if input.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if input.IdempotencyKey == "" {
		return nil, errors.New("idempotency key is required")
	}

	// 冪等性チェック
	existing, err := uc.repo.FindTransactionByIdempotencyKey(ctx, input.UserID, input.IdempotencyKey, domain.TransactionTypeRedeem)
	if err != nil {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}
	if existing != nil {
		if existing.Amount == input.Amount {
			account, err := uc.repo.GetAccount(ctx, input.UserID)
			if err != nil {
				return nil, fmt.Errorf("get account for idempotent response: %w", err)
			}
			return &RedeemOutput{
				TransactionID: existing.ID,
				Balance:       account.Balance,
				Version:       account.Version,
			}, nil
		}
		return nil, domain.ErrIdempotencyConflict
	}

	// 楽観ロック付きリトライ
	var output *RedeemOutput
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
		return nil, fmt.Errorf("redeem failed after %d retries: %w", maxRetries, err)
	}

	// キャッシュ無効化
	if invalidateErr := uc.cache.InvalidateBalance(ctx, input.UserID); invalidateErr != nil {
		uc.logger.Warn("failed to invalidate cache", zap.Error(invalidateErr))
	}

	uc.logger.Info("mileage redeemed",
		zap.String("user_id", input.UserID),
		zap.Int64("amount", input.Amount),
		zap.String("transaction_id", output.TransactionID),
	)

	return output, nil
}

func (uc *RedeemUsecase) executeInTx(ctx context.Context, input RedeemInput) (*RedeemOutput, error) {
	var output *RedeemOutput

	err := uc.txMgr.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. 口座取得
		account, err := uc.repo.GetAccount(txCtx, input.UserID)
		if err != nil {
			return fmt.Errorf("get account: %w", err)
		}
		if account == nil {
			return errors.New("account not found")
		}

		// 2. ドメインロジック（消費 + 残高不足チェック）
		expectedVersion := account.Version
		if err := account.Redeem(input.Amount); err != nil {
			return err
		}

		// 3. 取引記録
		txID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
		tx := &domain.MileageTransaction{
			ID:             txID,
			UserID:         input.UserID,
			Amount:         input.Amount,
			Type:           domain.TransactionTypeRedeem,
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

		output = &RedeemOutput{
			TransactionID: txID,
			Balance:       account.Balance,
			Version:       account.Version,
		}
		return nil
	})

	return output, err
}
