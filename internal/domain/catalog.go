package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	StatusActive      = "active"
	StatusMaintenance = "maintenance"
	StatusDisabled    = "disabled"
)

type Hotel struct {
	ID        int64
	Name      string
	Address   string
	CreatedAt time.Time
}

type RoomType struct {
	ID            int64
	HotelID       int64
	Name          string
	Capacity      int
	PricePerNight decimal.Decimal
	Active        bool
}

type Room struct {
	ID         int64
	HotelID    int64
	RoomTypeID int64
	Number     string
	Status     string
	CreatedAt  time.Time
}

type RoomView struct {
	Room
	TypeName      string
	PricePerNight decimal.Decimal
}

type CreateHotelInput struct {
	Name    string
	Address string
}

type CreateRoomTypeInput struct {
	Name          string
	Capacity      int
	PricePerNight decimal.Decimal
}

type UpdateRoomTypeInput struct {
	Name          *string
	Capacity      *int
	PricePerNight *decimal.Decimal
	Active        *bool
}

type CreateRoomInput struct {
	RoomTypeID int64
	Number     string
}
