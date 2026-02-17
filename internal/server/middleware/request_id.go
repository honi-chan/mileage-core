package middleware

import (
	"crypto/rand"

	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
)

const requestIDHeader = "X-Request-Id"

// RequestID はリクエストIDミドルウェア
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := c.Request().Header.Get(requestIDHeader)
			if reqID == "" {
				reqID = ulid.MustNew(ulid.Now(), rand.Reader).String()
			}
			c.Set("request_id", reqID)
			c.Response().Header().Set(requestIDHeader, reqID)
			return next(c)
		}
	}
}

// GetRequestID はコンテキストからリクエストIDを取得する
func GetRequestID(c echo.Context) string {
	v := c.Get("request_id")
	if v == nil {
		return ""
	}
	return v.(string)
}
