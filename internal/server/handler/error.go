package handler

import (
	"net/http"

	"github.com/honi-chan/mileage-core/internal/server/middleware"
	"github.com/labstack/echo/v4"
)

// ErrorResponse は統一エラーレスポンス
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody はエラーボディ
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// NewErrorResponse はエラーレスポンスを作成する
func NewErrorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: middleware.GetRequestID(c),
		},
	})
}

// BadRequest は400エラーを返す
func BadRequest(c echo.Context, message string) error {
	return NewErrorResponse(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// NotFound は404エラーを返す
func NotFound(c echo.Context, message string) error {
	return NewErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict は409エラーを返す
func Conflict(c echo.Context, code, message string) error {
	return NewErrorResponse(c, http.StatusConflict, code, message)
}

// InternalError は500エラーを返す
func InternalError(c echo.Context, message string) error {
	return NewErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}
