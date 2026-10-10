package repository

import (
	"context"
	"errors"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u domain.User) (domain.User, error) {
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (
			email,
			password_hash,
			name,
			phone,
			role
		)
		VALUES ($1, $2, $3, $4, 'guest')
		RETURNING id, role`,
		u.Email,
		u.PasswordHash,
		u.Name,
		u.Phone,
	).Scan(&u.ID, &u.Role)
	if err != nil {
		if pgCode(err) == pgUniqueViolation {
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, err
	}

	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	u := domain.User{Email: email}

	err := r.db.QueryRow(ctx,
		`SELECT id, password_hash, role
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}

	return u, nil
}
