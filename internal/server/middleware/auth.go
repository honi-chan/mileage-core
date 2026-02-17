package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// contextKey はコンテキストキーの型
type contextKey string

// UserIDKey はコンテキストからユーザーIDを取得するキー
const UserIDKey contextKey = "user_id"

// Auth はJWT認証ミドルウェア
func Auth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "UNAUTHORIZED",
						"message": "missing authorization header",
					},
				})
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "UNAUTHORIZED",
						"message": "invalid authorization header format",
					},
				})
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "UNAUTHORIZED",
						"message": "invalid or expired token",
					},
				})
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "UNAUTHORIZED",
						"message": "invalid token claims",
					},
				})
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "UNAUTHORIZED",
						"message": "invalid user id in token",
					},
				})
			}

			c.Set(string(UserIDKey), userID)
			return next(c)
		}
	}
}

// GetUserID はコンテキストからユーザーIDを取得する
func GetUserID(c echo.Context) string {
	v := c.Get(string(UserIDKey))
	if v == nil {
		return ""
	}
	return v.(string)
}
