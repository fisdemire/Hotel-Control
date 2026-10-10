package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users  userRepository
	secret []byte
	ttl    time.Duration
}

func NewAuthService(users userRepository, secret string, ttl time.Duration) *AuthService {
	return &AuthService{
		users:  users,
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (s *AuthService) Register(ctx context.Context, in domain.RegisterInput) (domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return domain.User{}, domain.ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}

	var phone *string
	if in.Phone != nil {
		if p := strings.TrimSpace(*in.Phone); p != "" {
			phone = &p
		}
	}

	return s.users.Create(ctx, domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(in.Name),
		Phone:        phone,
	})
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(u.PasswordHash),
		[]byte(password),
	); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	return s.issueToken(u)
}

func (s *AuthService) ParseToken(raw string) (domain.Principal, error) {
	var c claims

	token, err := jwt.ParseWithClaims(
		raw,
		&c,
		func(*jwt.Token) (any, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return domain.Principal{}, domain.ErrInvalidToken
	}

	userID, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return domain.Principal{}, domain.ErrInvalidToken
	}

	return domain.Principal{UserID: userID, Role: c.Role}, nil
}

func (s *AuthService) issueToken(u domain.User) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims{
			Role: u.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   strconv.FormatInt(u.ID, 10),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			},
		},
	)

	return token.SignedString(s.secret)
}
