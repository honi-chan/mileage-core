package mysql

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// NewDB はMySQL接続を初期化する
func NewDB(dsn string, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql: %w", err)
	}

	// 高負荷対応のための接続プール設定
	db.SetMaxOpenConns(100)            // 最大接続数を増加（25 → 100）
	db.SetMaxIdleConns(25)             // アイドル接続数を増加（10 → 25）
	db.SetConnMaxLifetime(3 * time.Minute) // 接続の最大生存時間（DB側のタイムアウト対策）
	db.SetConnMaxIdleTime(1 * time.Minute) // アイドル接続の最大時間（リソース効率化）

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	logger.Info("connected to MySQL",
		zap.Int("max_open_conns", 100),
		zap.Int("max_idle_conns", 25),
	)
	return db, nil
}
