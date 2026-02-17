package handler

import (
	"net/http"

	"github.com/honi-chan/mileage-core/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証ハンドラ
type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

// NewAuthHandler は新しいAuthHandlerを作成する
func NewAuthHandler(authUC *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// Signup はユーザー新規登録
func (h *AuthHandler) Signup(c echo.Context) error {
	var req signupRequest
	if err := c.Bind(&req); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest(c, "email and password are required")
	}
	if len(req.Password) < 8 {
		return BadRequest(c, "password must be at least 8 characters")
	}

	output, err := h.authUC.Signup(c.Request().Context(), usecase.SignupInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if err.Error() == "email already registered" {
			return Conflict(c, "EMAIL_CONFLICT", err.Error())
		}
		return InternalError(c, "signup failed")
	}

	return c.JSON(http.StatusCreated, authResponse{
		UserID: output.UserID,
		Token:  output.Token,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login はユーザーログイン
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return BadRequest(c, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest(c, "email and password are required")
	}

	output, err := h.authUC.Login(c.Request().Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if err.Error() == "invalid credentials" {
			return NewErrorResponse(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
		}
		return InternalError(c, "login failed")
	}

	return c.JSON(http.StatusOK, authResponse{
		UserID: output.UserID,
		Token:  output.Token,
	})
}
