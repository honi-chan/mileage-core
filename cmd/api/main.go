package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/honi-chan/mileage-core/internal/config"
	infraMySQL "github.com/honi-chan/mileage-core/internal/infra/mysql"
	infraRedis "github.com/honi-chan/mileage-core/internal/infra/redis"
	"github.com/honi-chan/mileage-core/internal/server"
	"github.com/honi-chan/mileage-core/internal/server/handler"
	"github.com/honi-chan/mileage-core/internal/usecase"
)

func main() {
	// Logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Config
	cfg := config.Load()

	// MySQL
	db, err := infraMySQL.NewDB(cfg.DB.DSN(), logger)
	if err != nil {
		logger.Fatal("failed to connect to MySQL", zap.Error(err))
	}
	defer db.Close()

	// Redis
	cache, err := infraRedis.NewCache(cfg.Redis.Addr, logger)
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer cache.Close()

	// Repositories
	userRepo := infraMySQL.NewUserRepository(db)
	mileageRepo := infraMySQL.NewMileageRepository(db)
	achievementRepo := infraMySQL.NewAchievementRepository(db)
	txManager := infraMySQL.NewTxManager(db)

	// Usecases
	authUC := usecase.NewAuthUsecase(userRepo, mileageRepo, cfg.JWT, logger)
	grantUC := usecase.NewGrantUsecase(mileageRepo, txManager, cache, logger)
	redeemUC := usecase.NewRedeemUsecase(mileageRepo, txManager, cache, logger)
	balanceUC := usecase.NewBalanceUsecase(mileageRepo, cache, logger)
	txUC := usecase.NewTransactionsUsecase(mileageRepo)
	achieveUC := usecase.NewAchievementUsecase(achievementRepo, logger)

	// Handlers
	authHandler := handler.NewAuthHandler(authUC)
	mileageHandler := handler.NewMileageHandler(grantUC, redeemUC, balanceUC, txUC)
	achievementHandler := handler.NewAchievementHandler(achieveUC)

	// Router
	e := server.NewRouter(cfg, logger, authHandler, mileageHandler, achievementHandler)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("shutting down server...")
		if err := e.Close(); err != nil {
			logger.Error("server shutdown error", zap.Error(err))
		}
	}()

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("starting server", zap.String("addr", addr))
	if err := e.Start(addr); err != nil {
		logger.Info("server stopped", zap.Error(err))
	}
}
