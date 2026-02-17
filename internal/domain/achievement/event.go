package achievement

import (
	"context"
	"time"
)

// AchievementEvent は行動イベント
type AchievementEvent struct {
	ID             string
	UserID         string
	ActionType     string
	IdempotencyKey string
	CreatedAt      time.Time
}

// AchievementRepository は行動イベントリポジトリインターフェース
type AchievementRepository interface {
	Create(ctx context.Context, event *AchievementEvent) error
}
