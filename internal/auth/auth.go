package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db     *pgxpool.Pool
	secret []byte
	ttl    time.Duration
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func New(
	db *pgxpool.Pool,
	secret string,
	ttl time.Duration,
) *Handler {
	return &Handler{
		db:     db,
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/auth/login", h.login)
	r.POST("/auth/register", h.register)
}

type registerRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8"`
	Name     string  `json:"name" binding:"required"`
	Phone    *string `json:"phone"`
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505"
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad request",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad request",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	var (
		id   int64
		role string
	)

	var phone any
	if req.Phone != nil {
		phone = strings.TrimSpace(*req.Phone)

		if phone == "" {
			phone = nil
		}
	}

	err = h.db.QueryRow(
		c.Request.Context(),
		`INSERT INTO users (
			email,
			password_hash,
			name,
			phone,
			role
		)
		VALUES ($1, $2, $3, $4, 'guest')
		RETURNING id, role`,
		email,
		hash,
		strings.TrimSpace(req.Name),
		phone,
	).Scan(&id, &role)

	if err != nil {
		if uniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    id,
		"email": email,
		"name":  strings.TrimSpace(req.Name),
		"phone": req.Phone,
		"role":  role,
	})
}

func (h *Handler) login(c *gin.Context) {
	var req credentialsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad request",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var (
		id   int64
		hash string
		role string
	)

	err := h.db.QueryRow(
		c.Request.Context(),
		`SELECT id, password_hash, role
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(&id, &hash, &role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "database error",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(req.Password),
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		Claims{
			Role: role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   strconv.FormatInt(id, 10),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.ttl)),
			},
		},
	)

	signedToken, err := token.SignedString(h.secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": signedToken,
	})
}

func Require(secret string, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		parts := strings.SplitN(header, " ", 2)

		if len(parts) != 2 ||
			parts[0] != "Bearer" ||
			parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header",
			})
			return
		}

		var claims Claims

		token, err := jwt.ParseWithClaims(
			parts[1],
			&claims,
			func(token *jwt.Token) (any, error) {
				return []byte(secret), nil
			},
			jwt.WithValidMethods([]string{
				jwt.SigningMethodHS256.Alg(),
			}),
			jwt.WithExpirationRequired(),
		)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || userID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		if len(roles) > 0 && !hasRole(claims.Role, roles) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden",
			})
			return
		}

		c.Set("user_id", userID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func hasRole(role string, roles []string) bool {
	for _, allowedRole := range roles {
		if role == allowedRole {
			return true
		}
	}

	return false
}
