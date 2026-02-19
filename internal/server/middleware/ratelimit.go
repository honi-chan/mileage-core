package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// RateLimit はIP単位のレート制限ミドルウェア
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	type limiterEntry struct {
		limiter  *rate.Limiter
		lastSeen int64 // Unix nanoタイムスタンプ（atomic操作用）
	}

	var (
		mu          sync.RWMutex
		limiters    = make(map[string]*limiterEntry)
		cleanupOnce sync.Once
	)

	// 定期的に古いエントリを削除（メモリリーク防止）
	// sync.Onceで一度だけ起動
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now().UnixNano()
				mu.Lock()
				for ip, entry := range limiters {
					lastSeen := atomic.LoadInt64(&entry.lastSeen)
					if now-lastSeen > int64(3*time.Minute) {
						delete(limiters, ip)
					}
				}
				mu.Unlock()
			}
		}()
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			now := time.Now().UnixNano()

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
						lastSeen: now,
					}
					limiters[ip] = entry
				}
				mu.Unlock()
			}

			// atomic操作でlastSeenを更新（ロック不要）
			atomic.StoreInt64(&entry.lastSeen, now)

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
