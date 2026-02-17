package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	domain "github.com/honi-chan/mileage-core/internal/domain/mileage"
	"github.com/honi-chan/mileage-core/internal/infra/redis"
)

// BalanceUsecase は残高照会ユースケース
type BalanceUsecase struct {
	repo   domain.MileageRepository
	cache  *redis.Cache
	logger *zap.Logger
}

// NewBalanceUsecase は新しいBalanceUsecaseを作成する
func NewBalanceUsecase(repo domain.MileageRepository, cache *redis.Cache, logger *zap.Logger) *BalanceUsecase {
	return &BalanceUsecase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// BalanceOutput は残高出力
type BalanceOutput struct {
	UserID  string
	Balance int64
	Version int64
}

// Execute は残高を取得する（Redis → DB フォールバック）
func (uc *BalanceUsecase) Execute(ctx context.Context, userID string) (*BalanceOutput, error) {
	// Redisキャッシュを先にチェック
	cached, err := uc.cache.GetBalance(ctx, userID)
	if err != nil {
		uc.logger.Warn("cache read failed, falling back to DB", zap.Error(err))
	}
	if cached != nil {
		return &BalanceOutput{
			UserID:  userID,
			Balance: cached.Balance,
			Version: cached.Version,
		}, nil
	}

	// キャッシュミス → DBから取得
	account, err := uc.repo.GetAccount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found")
	}

	// キャッシュに書き込み
	if setErr := uc.cache.SetBalance(ctx, userID, account.Balance, account.Version); setErr != nil {
		uc.logger.Warn("cache write failed", zap.Error(setErr))
	}

	return &BalanceOutput{
		UserID:  userID,
		Balance: account.Balance,
		Version: account.Version,
	}, nil
}
