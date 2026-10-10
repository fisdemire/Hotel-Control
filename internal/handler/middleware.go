package handler

import (
	"net/http"
	"strings"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/gin-gonic/gin"
)

type Middlewares struct {
	Admin gin.HandlerFunc
	Staff gin.HandlerFunc
}

type tokenParser interface {
	ParseToken(raw string) (domain.Principal, error)
}

func RequireRoles(parser tokenParser, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header",
			})
			return
		}

		p, err := parser.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		if len(roles) > 0 && !hasRole(p.Role, roles) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden",
			})
			return
		}

		c.Set("user_id", p.UserID)
		c.Set("role", p.Role)

		c.Next()
	}
}

func hasRole(role string, roles []string) bool {
	for _, allowed := range roles {
		if role == allowed {
			return true
		}
	}

	return false
}
