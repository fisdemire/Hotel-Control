package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, err error) {
	var inputErr *domain.InputError

	switch {
	case errors.As(err, &inputErr):
		badRequest(c, inputErr.Msg)

	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidReference):
		badRequest(c, err.Error())

	case errors.Is(err, domain.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

	case errors.Is(err, domain.ErrHotelNotFound),
		errors.Is(err, domain.ErrRoomTypeNotFound),
		errors.Is(err, domain.ErrRoomNotFound),
		errors.Is(err, domain.ErrRoomUnavailable),
		errors.Is(err, domain.ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrRoomTypeExists),
		errors.Is(err, domain.ErrRoomNumberExists),
		errors.Is(err, domain.ErrAlreadyBooked):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		log.Printf("handler: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func pathID(c *gin.Context, msg string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, msg)
		return 0, false
	}

	return id, true
}

func mapSlice[T, R any](in []T, f func(T) R) []R {
	out := make([]R, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}

	return out
}
