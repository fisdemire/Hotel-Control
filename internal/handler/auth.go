package handler

import (
	"context"
	"net/http"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/gin-gonic/gin"
)

type authService interface {
	Register(ctx context.Context, in domain.RegisterInput) (domain.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type AuthHandler struct {
	svc authService
}

func NewAuthHandler(svc authService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Routes(r gin.IRouter) {
	r.POST("/auth/login", h.login)
	r.POST("/auth/register", h.register)
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8"`
	Name     string  `json:"name" binding:"required"`
	Phone    *string `json:"phone"`
}

func (h *AuthHandler) register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	user, err := h.svc.Register(c.Request.Context(), domain.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Phone:    req.Phone,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) login(c *gin.Context) {
	var req credentialsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	token, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
