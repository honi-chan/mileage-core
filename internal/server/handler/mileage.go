package handler

import (
	"errors"
	"net/http"
	"strconv"

	domain "github.com/honi-chan/mileage-core/internal/domain/mileage"
	"github.com/honi-chan/mileage-core/internal/server/middleware"
	"github.com/honi-chan/mileage-core/internal/usecase"
	"github.com/labstack/echo/v4"
)

// MileageHandler はマイレージハンドラ
type MileageHandler struct {
	grantUC   *usecase.GrantUsecase
	redeemUC  *usecase.RedeemUsecase
	balanceUC *usecase.BalanceUsecase
	txUC      *usecase.TransactionsUsecase
}

// NewMileageHandler は新しいMileageHandlerを作成する
func NewMileageHandler(
	grantUC *usecase.GrantUsecase,
	redeemUC *usecase.RedeemUsecase,
	balanceUC *usecase.BalanceUsecase,
	txUC *usecase.TransactionsUsecase,
) *MileageHandler {
	return &MileageHandler{
		grantUC:   grantUC,
		redeemUC:  redeemUC,
		balanceUC: balanceUC,
		txUC:      txUC,
	}
}

type balanceResponse struct {
	UserID  string `json:"user_id"`
	Balance int64  `json:"balance"`
	Version int64  `json:"version"`
}

// GetBalance はマイル残高を取得する
func (h *MileageHandler) GetBalance(c echo.Context) error {
	userID := middleware.GetUserID(c)

	output, err := h.balanceUC.Execute(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get balance")
	}

	return c.JSON(http.StatusOK, balanceResponse{
		UserID:  output.UserID,
		Balance: output.Balance,
		Version: output.Version,
	})
}

// GetTransactions は取引履歴を取得する
func (h *MileageHandler) GetTransactions(c echo.Context) error {
	userID := middleware.GetUserID(c)
	cursor := c.QueryParam("cursor")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 50
	}

	output, err := h.txUC.Execute(c.Request().Context(), usecase.TransactionsInput{
		UserID: userID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		return InternalError(c, "failed to get transactions")
	}

	return c.JSON(http.StatusOK, output)
}

type grantRequest struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

type grantResponse struct {
	TransactionID string `json:"transaction_id"`
	Balance       int64  `json:"balance"`
	Version       int64  `json:"version"`
}

// Grant はマイルを付与する
func (h *MileageHandler) Grant(c echo.Context) error {
	userID := middleware.GetUserID(c)
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return BadRequest(c, "Idempotency-Key header is required")
	}

	var req grantRequest
	if err := c.Bind(&req); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if req.Amount <= 0 {
		return BadRequest(c, "amount must be positive")
	}

	output, err := h.grantUC.Execute(c.Request().Context(), usecase.GrantInput{
		UserID:         userID,
		Amount:         req.Amount,
		Reason:         req.Reason,
		IdempotencyKey: idempotencyKey,
		RequestID:      middleware.GetRequestID(c),
	})
	if err != nil {
		return handleMileageError(c, err, "grant")
	}

	middleware.RecordBusinessOp("grant", "success")
	return c.JSON(http.StatusOK, grantResponse{
		TransactionID: output.TransactionID,
		Balance:       output.Balance,
		Version:       output.Version,
	})
}

type redeemRequest struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

type redeemResponse struct {
	TransactionID string `json:"transaction_id"`
	Balance       int64  `json:"balance"`
	Version       int64  `json:"version"`
}

// Redeem はマイルを消費する
func (h *MileageHandler) Redeem(c echo.Context) error {
	userID := middleware.GetUserID(c)
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return BadRequest(c, "Idempotency-Key header is required")
	}

	var req redeemRequest
	if err := c.Bind(&req); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if req.Amount <= 0 {
		return BadRequest(c, "amount must be positive")
	}

	output, err := h.redeemUC.Execute(c.Request().Context(), usecase.RedeemInput{
		UserID:         userID,
		Amount:         req.Amount,
		Reason:         req.Reason,
		IdempotencyKey: idempotencyKey,
		RequestID:      middleware.GetRequestID(c),
	})
	if err != nil {
		return handleMileageError(c, err, "redeem")
	}

	middleware.RecordBusinessOp("redeem", "success")
	return c.JSON(http.StatusOK, redeemResponse{
		TransactionID: output.TransactionID,
		Balance:       output.Balance,
		Version:       output.Version,
	})
}

func handleMileageError(c echo.Context, err error, operation string) error {
	switch {
	case errors.Is(err, domain.ErrInsufficientBalance):
		middleware.RecordBusinessOp(operation, "insufficient_balance")
		return Conflict(c, "INSUFFICIENT_BALANCE", "balance is not enough")
	case errors.Is(err, domain.ErrOptimisticLock):
		middleware.RecordBusinessOp(operation, "optimistic_lock")
		return Conflict(c, "OPTIMISTIC_LOCK_CONFLICT", "concurrent update detected, please retry")
	case errors.Is(err, domain.ErrIdempotencyConflict):
		middleware.RecordBusinessOp(operation, "idempotency_conflict")
		return Conflict(c, "IDEMPOTENCY_CONFLICT", "same idempotency key with different content")
	case errors.Is(err, domain.ErrInvalidAmount):
		return BadRequest(c, "amount must be positive")
	default:
		middleware.RecordBusinessOp(operation, "error")
		return InternalError(c, operation+" failed")
	}
}
