package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type CreateBookingInput struct {
	RoomID     int64
	GuestName  string
	GuestEmail string
	GuestPhone *string
	CheckIn    time.Time
	CheckOut   time.Time
}

type AvailabilityQuery struct {
	HotelID int64
	From    time.Time
	To      time.Time
	Guests  int
}

type BookingFilter struct {
	HotelID *int64
	Status  *string
	From    *time.Time
	To      *time.Time
}

type AvailableRoom struct {
	RoomID        int64           `json:"room_id"`
	Number        string          `json:"number"`
	TypeName      string          `json:"type_name"`
	Capacity      int             `json:"capacity"`
	PricePerNight decimal.Decimal `json:"price_per_night"`
	TotalPrice    decimal.Decimal `json:"total_price"`
}

type Booking struct {
	ID            int64           `json:"id"`
	RoomID        int64           `json:"room_id"`
	RoomNumber    string          `json:"room_number"`
	HotelID       int64           `json:"hotel_id"`
	GuestName     string          `json:"guest_name"`
	GuestEmail    string          `json:"guest_email"`
	GuestPhone    *string         `json:"guest_phone,omitempty"`
	CheckIn       string          `json:"check_in"`
	CheckOut      string          `json:"check_out"`
	Status        string          `json:"status"`
	PricePerNight decimal.Decimal `json:"price_per_night"`
	TotalPrice    decimal.Decimal `json:"total_price"`
	CreatedAt     time.Time       `json:"created_at"`
}
