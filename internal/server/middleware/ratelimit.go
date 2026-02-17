package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimit はIP単位のレート制限ミドルウェア
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !limiter.Allow() {
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "too many requests",
					},
				})
			}
			return next(c)
		}
	}
}
