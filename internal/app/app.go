package app

import (
	"net/http"

	"github.com/fisdemire/Hotel-Control/config"
	"github.com/fisdemire/Hotel-Control/internal/auth"
	"github.com/fisdemire/Hotel-Control/internal/booking"
	"github.com/fisdemire/Hotel-Control/internal/rooms"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg *config.Config
	db  *pgxpool.Pool
}

func New(cfg *config.Config, db *pgxpool.Pool) *App {
	return &App{
		cfg: cfg,
		db:  db,
	}
}

func (a *App) Run() error {
	r := gin.Default()

	authHandler := auth.New(
		a.db,
		a.cfg.Auth.JWTSecret,
		a.cfg.Auth.TokenTTL,
	)

	authHandler.RegisterRoutes(r)

	admin := auth.Require(
		a.cfg.Auth.JWTSecret,
		"admin",
	)

	staff := auth.Require(
		a.cfg.Auth.JWTSecret,
		"admin",
		"manager",
	)

	rooms.Register(
		r,
		a.db,
		admin,
		staff,
	)

	booking.Register(
		r,
		a.db,
		staff,
		admin,
	)

	r.GET("/health", func(c *gin.Context) {
		if err := a.db.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db is down",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	srv := &http.Server{
		Addr:         a.cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
		IdleTimeout:  a.cfg.Server.IdleTimeout,
	}

	return srv.ListenAndServe()
}
