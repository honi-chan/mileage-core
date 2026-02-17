package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/honi-chan/mileage-core/internal/domain/user"
	"github.com/jmoiron/sqlx"
)

// UserRepository はユーザーリポジトリのMySQL実装
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository は新しいUserRepositoryを作成する
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

type userRow struct {
	ID           string       `db:"id"`
	Email        string       `db:"email"`
	PasswordHash string       `db:"password_hash"`
	CreatedAt    sql.NullTime `db:"created_at"`
	UpdatedAt    sql.NullTime `db:"updated_at"`
}

func (r *userRow) toDomain() *user.User {
	u := &user.User{
		ID:           r.ID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
	}
	if r.CreatedAt.Valid {
		u.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		u.UpdatedAt = r.UpdatedAt.Time
	}
	return u
}

func (repo *UserRepository) Create(ctx context.Context, u *user.User) error {
	q := GetQuerier(ctx, repo.db)
	_, err := q.ExecContext(ctx,
		"INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)",
		u.ID, u.Email, u.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	q := GetQuerier(ctx, repo.db)
	var row userRow
	err := q.GetContext(ctx, &row, "SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?", email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return row.toDomain(), nil
}

func (repo *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	q := GetQuerier(ctx, repo.db)
	var row userRow
	err := q.GetContext(ctx, &row, "SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return row.toDomain(), nil
}
