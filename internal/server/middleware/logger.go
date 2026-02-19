package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Logger は構造化ログミドルウェア
func Logger(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			latency := time.Since(start)

			fields := []zap.Field{
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", latency),
				zap.String("request_id", GetRequestID(c)),
				zap.String("remote_ip", c.RealIP()),
			}

			if userID := GetUserID(c); userID != "" {
				fields = append(fields, zap.String("user_id", userID))
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error("request failed", fields...)
			} else if c.Response().Status >= 400 {
				logger.Warn("request completed with error status", fields...)
			} else {
				logger.Info("request completed", fields...)
			}

			return err
		}
	}
}
