package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const dateLayout = "2006-01-02"

type bookingService interface {
	Create(ctx context.Context, in domain.CreateBookingInput) (int64, decimal.Decimal, error)
	Available(ctx context.Context, q domain.AvailabilityQuery) ([]domain.AvailableRoom, error)
	List(ctx context.Context, f domain.BookingFilter) ([]domain.Booking, error)
	Cancel(ctx context.Context, id int64) error
}

type BookingHandler struct {
	svc bookingService
}

func NewBookingHandler(svc bookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) Routes(r gin.IRouter, mw Middlewares) {
	r.GET("/hotels/:id/availability", h.availability)
	r.POST("/bookings", h.createBooking)
	r.GET("/bookings", mw.Staff, h.listBookings)
	r.POST("/bookings/:id/cancel", mw.Staff, h.cancelBooking)
}

type createBookingRequest struct {
	RoomID     int64   `json:"room_id" binding:"required"`
	GuestName  string  `json:"guest_name" binding:"required"`
	GuestEmail string  `json:"guest_email" binding:"required,email"`
	GuestPhone *string `json:"guest_phone"`
	CheckIn    string  `json:"check_in" binding:"required"`
	CheckOut   string  `json:"check_out" binding:"required"`
}

type createBookingResponse struct {
	ID         int64           `json:"id"`
	TotalPrice decimal.Decimal `json:"total_price"`
}

func (h *BookingHandler) availability(c *gin.Context) {
	hotelID, ok := pathID(c, "invalid hotel id")
	if !ok {
		return
	}

	from, err := parseDate(c.Query("from"))
	if err != nil {
		badRequest(c, "invalid from date")
		return
	}

	to, err := parseDate(c.Query("to"))
	if err != nil {
		badRequest(c, "invalid to date")
		return
	}

	guests := 1

	if value := c.Query("guests"); value != "" {
		guests, err = strconv.Atoi(value)
		if err != nil {
			badRequest(c, "guests must be a positive integer")
			return
		}
	}

	rooms, err := h.svc.Available(c.Request.Context(), domain.AvailabilityQuery{
		HotelID: hotelID,
		From:    from,
		To:      to,
		Guests:  guests,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *BookingHandler) createBooking(c *gin.Context) {
	var req createBookingRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	checkIn, err := parseDate(req.CheckIn)
	if err != nil {
		badRequest(c, "invalid check_in date")
		return
	}

	checkOut, err := parseDate(req.CheckOut)
	if err != nil {
		badRequest(c, "invalid check_out date")
		return
	}

	id, total, err := h.svc.Create(c.Request.Context(), domain.CreateBookingInput{
		RoomID:     req.RoomID,
		GuestName:  req.GuestName,
		GuestEmail: req.GuestEmail,
		GuestPhone: req.GuestPhone,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createBookingResponse{
		ID:         id,
		TotalPrice: total,
	})
}

func (h *BookingHandler) listBookings(c *gin.Context) {
	var filter domain.BookingFilter
	var err error

	if value := c.Query("hotel_id"); value != "" {
		hotelID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			badRequest(c, "invalid hotel_id")
			return
		}

		filter.HotelID = &hotelID
	}

	if value := c.Query("status"); value != "" {
		filter.Status = &value
	}

	filter.From, err = optionalDate(c.Query("from"))
	if err != nil {
		badRequest(c, "invalid from date")
		return
	}

	filter.To, err = optionalDate(c.Query("to"))
	if err != nil {
		badRequest(c, "invalid to date")
		return
	}

	bookings, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, bookings)
}

func (h *BookingHandler) cancelBooking(c *gin.Context) {
	id, ok := pathID(c, "invalid booking id")
	if !ok {
		return
	}

	if err := h.svc.Cancel(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "cancelled",
	})
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, value, time.UTC)
}

func optionalDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	t, err := parseDate(value)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
