package server

import (
	"net/http/pprof"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/honi-chan/mileage-core/internal/config"
	"github.com/honi-chan/mileage-core/internal/server/handler"
	"github.com/honi-chan/mileage-core/internal/server/middleware"
)

// NewRouter はEchoルーターを設定する
func NewRouter(
	cfg *config.Config,
	logger *zap.Logger,
	authHandler *handler.AuthHandler,
	mileageHandler *handler.MileageHandler,
	achievementHandler *handler.AchievementHandler,
) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// グローバルミドルウェア
	e.Use(echoMiddleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger(logger))
	e.Use(middleware.Metrics())
	e.Use(middleware.RateLimit(cfg.RateLimit.RPS, cfg.RateLimit.Burst))
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-Id"},
	}))

	// ヘルスチェック
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// メトリクス
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// pprof（本番は制限すること）
	pprofGroup := e.Group("/debug/pprof")
	pprofGroup.GET("/", echo.WrapHandler(pprof.Handler("index")))
	pprofGroup.GET("/cmdline", echo.WrapHandler(pprof.Handler("cmdline")))
	pprofGroup.GET("/profile", echo.WrapHandler(pprof.Handler("profile")))
	pprofGroup.GET("/symbol", echo.WrapHandler(pprof.Handler("symbol")))
	pprofGroup.GET("/trace", echo.WrapHandler(pprof.Handler("trace")))
	pprofGroup.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
	pprofGroup.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
	pprofGroup.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))

	// Auth API（認証不要）
	v1 := e.Group("/v1")
	v1.POST("/auth/signup", authHandler.Signup)
	v1.POST("/auth/login", authHandler.Login)

	// 認証が必要なAPI
	authGroup := v1.Group("", middleware.Auth(cfg.JWT.Secret))

	// Mileage API
	authGroup.GET("/mileage/balance", mileageHandler.GetBalance)
	authGroup.GET("/mileage/transactions", mileageHandler.GetTransactions)
	authGroup.POST("/mileage/grant", mileageHandler.Grant)
	authGroup.POST("/mileage/redeem", mileageHandler.Redeem)

	// Achievement API
	authGroup.POST("/achievements", achievementHandler.Create)

	return e
}
