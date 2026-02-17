package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"

	"github.com/honi-chan/mileage-core/internal/domain/achievement"
)

// AchievementUsecase は行動イベントユースケース
type AchievementUsecase struct {
	repo   achievement.AchievementRepository
	logger *zap.Logger
}

// NewAchievementUsecase は新しいAchievementUsecaseを作成する
func NewAchievementUsecase(repo achievement.AchievementRepository, logger *zap.Logger) *AchievementUsecase {
	return &AchievementUsecase{repo: repo, logger: logger}
}

// AchievementInput は行動イベント入力
type AchievementInput struct {
	UserID         string
	ActionType     string
	IdempotencyKey string
}

// Execute は行動イベントを記録する
func (uc *AchievementUsecase) Execute(ctx context.Context, input AchievementInput) error {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()

	event := &achievement.AchievementEvent{
		ID:             id,
		UserID:         input.UserID,
		ActionType:     input.ActionType,
		IdempotencyKey: input.IdempotencyKey,
	}

	if err := uc.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("create achievement event: %w", err)
	}

	uc.logger.Info("achievement recorded",
		zap.String("user_id", input.UserID),
		zap.String("action_type", input.ActionType),
	)
	return nil
}
