package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/fisdemire/Hotel-Control/config"
	"github.com/fisdemire/Hotel-Control/internal/handler"
	"github.com/fisdemire/Hotel-Control/internal/repository"
	"github.com/fisdemire/Hotel-Control/internal/service"
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

func (a *App) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:         a.cfg.Server.Addr,
		Handler:      a.router(),
		ReadTimeout:  a.cfg.Server.ReadTimeout,
		WriteTimeout: a.cfg.Server.WriteTimeout,
		IdleTimeout:  a.cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)

	log.Printf("listening on %s", srv.Addr)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// сервер не смог стартовать или упал сам
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}

func (a *App) router() *gin.Engine {
	r := gin.Default()

	userRepo := repository.NewUserRepository(a.db)
	catalogRepo := repository.NewCatalogRepository(a.db)
	bookingRepo := repository.NewBookingRepository(a.db)

	authSvc := service.NewAuthService(userRepo, a.cfg.Auth.JWTSecret, a.cfg.Auth.TokenTTL)
	catalogSvc := service.NewCatalogService(catalogRepo)
	bookingSvc := service.NewBookingService(bookingRepo, time.UTC)

	mw := handler.Middlewares{
		Admin: handler.RequireRoles(authSvc, "admin"),
		Staff: handler.RequireRoles(authSvc, "admin", "manager"),
	}

	handler.NewAuthHandler(authSvc).Routes(r)
	handler.NewCatalogHandler(catalogSvc).Routes(r, mw)
	handler.NewBookingHandler(bookingSvc).Routes(r, mw)

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

	return r
}
