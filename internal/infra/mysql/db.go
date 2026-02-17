package mysql

import (
	"context"
	"fmt"

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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	logger.Info("connected to MySQL")
	return db, nil
}
