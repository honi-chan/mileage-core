package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimit はIP単位のレート制限ミドルウェア
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	type limiterEntry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu       sync.RWMutex
		limiters = make(map[string]*limiterEntry)
	)

	// 定期的に古いエントリを削除（メモリリーク防止）
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, entry := range limiters {
				if time.Since(entry.lastSeen) > 3*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			mu.RLock()
			entry, exists := limiters[ip]
			mu.RUnlock()

			if !exists {
				mu.Lock()
				// Double-check after acquiring write lock
				entry, exists = limiters[ip]
				if !exists {
					entry = &limiterEntry{
						limiter:  rate.NewLimiter(rate.Limit(rps), burst),
						lastSeen: time.Now(),
					}
					limiters[ip] = entry
				}
				mu.Unlock()
			}

			entry.lastSeen = time.Now()

			if !entry.limiter.Allow() {
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
