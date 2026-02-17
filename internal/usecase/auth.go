package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/honi-chan/mileage-core/internal/config"
	"github.com/honi-chan/mileage-core/internal/domain/user"
)

// AuthUsecase は認証ユースケース
type AuthUsecase struct {
	userRepo       user.UserRepository
	mileageAccount MileageAccountCreator
	jwtConfig      config.JWTConfig
	logger         *zap.Logger
}

// MileageAccountCreator はマイレージ口座作成のインターフェース
type MileageAccountCreator interface {
	CreateAccount(ctx context.Context, userID string) error
}

// NewAuthUsecase は新しいAuthUsecaseを作成する
func NewAuthUsecase(userRepo user.UserRepository, mileageCreator MileageAccountCreator, jwtConfig config.JWTConfig, logger *zap.Logger) *AuthUsecase {
	return &AuthUsecase{
		userRepo:       userRepo,
		mileageAccount: mileageCreator,
		jwtConfig:      jwtConfig,
		logger:         logger,
	}
}

// SignupInput はサインアップ入力
type SignupInput struct {
	Email    string
	Password string
}

// AuthOutput は認証出力
type AuthOutput struct {
	UserID string
	Token  string
}

// Signup はユーザーを新規登録する
func (uc *AuthUsecase) Signup(ctx context.Context, input SignupInput) (*AuthOutput, error) {
	// メールアドレスの重複チェック
	existing, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// パスワードハッシュ
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// ULID生成
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()

	u := &user.User{
		ID:           id,
		Email:        input.Email,
		PasswordHash: string(hash),
	}

	if err := uc.userRepo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// マイレージ口座を同時作成
	if err := uc.mileageAccount.CreateAccount(ctx, id); err != nil {
		return nil, fmt.Errorf("create mileage account: %w", err)
	}

	// JWT生成
	token, err := uc.generateToken(id)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	uc.logger.Info("user signed up", zap.String("user_id", id))
	return &AuthOutput{UserID: id, Token: token}, nil
}

// LoginInput はログイン入力
type LoginInput struct {
	Email    string
	Password string
}

// Login はユーザーをログインさせる
func (uc *AuthUsecase) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	u, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if u == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := uc.generateToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	uc.logger.Info("user logged in", zap.String("user_id", u.ID))
	return &AuthOutput{UserID: u.ID, Token: token}, nil
}

func (uc *AuthUsecase) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(uc.jwtConfig.ExpiryTime).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtConfig.Secret))
}

// GenerateNonce はランダムなノンスを生成する（ユーティリティ）
func GenerateNonce() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
