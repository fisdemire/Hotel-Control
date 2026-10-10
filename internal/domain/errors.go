package domain

import "errors"

var (
	ErrInvalidInput = errors.New("bad request")

	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrEmailTaken         = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")

	ErrHotelNotFound    = errors.New("hotel not found")
	ErrRoomTypeNotFound = errors.New("room type not found")
	ErrRoomNotFound     = errors.New("room not found")
	ErrRoomTypeExists   = errors.New("room type already exists")
	ErrRoomNumberExists = errors.New("room number already exists")
	ErrInvalidReference = errors.New("invalid room type or hotel")

	ErrRoomUnavailable = errors.New("room not found or unavailable")
	ErrAlreadyBooked   = errors.New("room is already booked for these dates")
	ErrBookingNotFound = errors.New("booking not found or already cancelled")
)

type InputError struct{ Msg string }

func (e *InputError) Error() string { return e.Msg }

func NewInputError(msg string) error { return &InputError{Msg: msg} }
