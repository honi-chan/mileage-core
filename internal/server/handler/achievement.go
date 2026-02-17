package handler

import (
	"net/http"

	"github.com/honi-chan/mileage-core/internal/server/middleware"
	"github.com/honi-chan/mileage-core/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AchievementHandler は行動イベントハンドラ
type AchievementHandler struct {
	achieveUC *usecase.AchievementUsecase
}

// NewAchievementHandler は新しいAchievementHandlerを作成する
func NewAchievementHandler(achieveUC *usecase.AchievementUsecase) *AchievementHandler {
	return &AchievementHandler{achieveUC: achieveUC}
}

type achievementRequest struct {
	ActionType     string `json:"action_type"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Create は行動イベントを登録する
func (h *AchievementHandler) Create(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var req achievementRequest
	if err := c.Bind(&req); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if req.ActionType == "" {
		return BadRequest(c, "action_type is required")
	}

	err := h.achieveUC.Execute(c.Request().Context(), usecase.AchievementInput{
		UserID:         userID,
		ActionType:     req.ActionType,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return InternalError(c, "failed to record achievement")
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}
