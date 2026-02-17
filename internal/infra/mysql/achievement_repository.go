package mysql

import (
	"context"
	"fmt"

	"github.com/honi-chan/mileage-core/internal/domain/achievement"
	"github.com/jmoiron/sqlx"
)

// AchievementRepository は行動イベントリポジトリのMySQL実装
type AchievementRepository struct {
	db *sqlx.DB
}

// NewAchievementRepository は新しいAchievementRepositoryを作成する
func NewAchievementRepository(db *sqlx.DB) *AchievementRepository {
	return &AchievementRepository{db: db}
}

func (repo *AchievementRepository) Create(ctx context.Context, event *achievement.AchievementEvent) error {
	q := GetQuerier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		"INSERT INTO achievement_events (id, user_id, action_type, idempotency_key) VALUES (?, ?, ?, ?)",
		event.ID, event.UserID, event.ActionType, event.IdempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("insert achievement event: %w", err)
	}
	return nil
}
