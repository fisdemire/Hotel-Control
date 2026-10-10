package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type catalogService interface {
	ListHotels(ctx context.Context) ([]domain.Hotel, error)
	CreateHotel(ctx context.Context, in domain.CreateHotelInput) (domain.Hotel, error)

	ListRoomTypes(ctx context.Context, hotelID int64) ([]domain.RoomType, error)
	CreateRoomType(ctx context.Context, hotelID int64, in domain.CreateRoomTypeInput) (domain.RoomType, error)
	UpdateRoomType(ctx context.Context, id int64, in domain.UpdateRoomTypeInput) (domain.RoomType, error)

	ListRooms(ctx context.Context, hotelID int64) ([]domain.RoomView, error)
	CreateRoom(ctx context.Context, hotelID int64, in domain.CreateRoomInput) (domain.Room, error)
	UpdateRoomStatus(ctx context.Context, id int64, status string) (domain.Room, error)
}

type CatalogHandler struct {
	svc catalogService
}

func NewCatalogHandler(svc catalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) Routes(r gin.IRouter, mw Middlewares) {
	r.GET("/hotels", h.listHotels)
	r.POST("/hotels", mw.Admin, h.createHotel)

	r.GET("/hotels/:id/room-types", h.listRoomTypes)
	r.POST("/hotels/:id/room-types", mw.Admin, h.createRoomType)
	r.PATCH("/room-types/:id", mw.Admin, h.updateRoomType)

	r.GET("/hotels/:id/rooms", h.listRooms)
	r.POST("/hotels/:id/rooms", mw.Staff, h.createRoom)
	r.PATCH("/rooms/:id/status", mw.Staff, h.updateRoomStatus)
}

type createHotelRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
}

type createRoomTypeRequest struct {
	Name          string          `json:"name" binding:"required"`
	Capacity      int             `json:"capacity" binding:"required"`
	PricePerNight decimal.Decimal `json:"price_per_night" binding:"required"`
}

type updateRoomTypeRequest struct {
	Name          *string          `json:"name"`
	Capacity      *int             `json:"capacity"`
	PricePerNight *decimal.Decimal `json:"price_per_night"`
	Active        *bool            `json:"active"`
}

type createRoomRequest struct {
	RoomTypeID int64  `json:"room_type_id" binding:"required"`
	Number     string `json:"number" binding:"required"`
}

type updateRoomStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type hotelResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type roomTypeResponse struct {
	ID            int64           `json:"id"`
	HotelID       int64           `json:"hotel_id"`
	Name          string          `json:"name"`
	Capacity      int             `json:"capacity"`
	PricePerNight decimal.Decimal `json:"price_per_night"`
	Active        bool            `json:"active"`
}

type roomResponse struct {
	ID         int64     `json:"id"`
	HotelID    int64     `json:"hotel_id"`
	RoomTypeID int64     `json:"room_type_id"`
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type roomViewResponse struct {
	roomResponse
	TypeName      string          `json:"type_name"`
	PricePerNight decimal.Decimal `json:"price_per_night"`
}

func newHotelResponse(h domain.Hotel) hotelResponse {
	return hotelResponse{ID: h.ID, Name: h.Name, Address: h.Address, CreatedAt: h.CreatedAt}
}

func newRoomTypeResponse(rt domain.RoomType) roomTypeResponse {
	return roomTypeResponse{
		ID:            rt.ID,
		HotelID:       rt.HotelID,
		Name:          rt.Name,
		Capacity:      rt.Capacity,
		PricePerNight: rt.PricePerNight,
		Active:        rt.Active,
	}
}

func newRoomResponse(r domain.Room) roomResponse {
	return roomResponse{
		ID:         r.ID,
		HotelID:    r.HotelID,
		RoomTypeID: r.RoomTypeID,
		Number:     r.Number,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt,
	}
}

func newRoomViewResponse(v domain.RoomView) roomViewResponse {
	return roomViewResponse{
		roomResponse:  newRoomResponse(v.Room),
		TypeName:      v.TypeName,
		PricePerNight: v.PricePerNight,
	}
}

func (h *CatalogHandler) listHotels(c *gin.Context) {
	hotels, err := h.svc.ListHotels(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapSlice(hotels, newHotelResponse))
}

func (h *CatalogHandler) createHotel(c *gin.Context) {
	var req createHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	hotel, err := h.svc.CreateHotel(c.Request.Context(), domain.CreateHotelInput{
		Name:    req.Name,
		Address: req.Address,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newHotelResponse(hotel))
}

func (h *CatalogHandler) listRoomTypes(c *gin.Context) {
	hotelID, ok := pathID(c, "invalid hotel id")
	if !ok {
		return
	}

	types, err := h.svc.ListRoomTypes(c.Request.Context(), hotelID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapSlice(types, newRoomTypeResponse))
}

func (h *CatalogHandler) createRoomType(c *gin.Context) {
	hotelID, ok := pathID(c, "invalid hotel id")
	if !ok {
		return
	}

	var req createRoomTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	rt, err := h.svc.CreateRoomType(c.Request.Context(), hotelID, domain.CreateRoomTypeInput{
		Name:          req.Name,
		Capacity:      req.Capacity,
		PricePerNight: req.PricePerNight,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newRoomTypeResponse(rt))
}

func (h *CatalogHandler) updateRoomType(c *gin.Context) {
	id, ok := pathID(c, "invalid room type id")
	if !ok {
		return
	}

	var req updateRoomTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	rt, err := h.svc.UpdateRoomType(c.Request.Context(), id, domain.UpdateRoomTypeInput{
		Name:          req.Name,
		Capacity:      req.Capacity,
		PricePerNight: req.PricePerNight,
		Active:        req.Active,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newRoomTypeResponse(rt))
}

func (h *CatalogHandler) listRooms(c *gin.Context) {
	hotelID, ok := pathID(c, "invalid hotel id")
	if !ok {
		return
	}

	rooms, err := h.svc.ListRooms(c.Request.Context(), hotelID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapSlice(rooms, newRoomViewResponse))
}

func (h *CatalogHandler) createRoom(c *gin.Context) {
	hotelID, ok := pathID(c, "invalid hotel id")
	if !ok {
		return
	}

	var req createRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	room, err := h.svc.CreateRoom(c.Request.Context(), hotelID, domain.CreateRoomInput{
		RoomTypeID: req.RoomTypeID,
		Number:     req.Number,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newRoomResponse(room))
}

func (h *CatalogHandler) updateRoomStatus(c *gin.Context) {
	id, ok := pathID(c, "invalid room id")
	if !ok {
		return
	}

	var req updateRoomStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "bad request")
		return
	}

	room, err := h.svc.UpdateRoomStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, newRoomResponse(room))
}
