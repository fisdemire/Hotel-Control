package service

import (
	"context"
	"strings"
	"time"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/shopspring/decimal"
)

type bookingRepository interface {
	Insert(ctx context.Context, in domain.CreateBookingInput) (int64, decimal.Decimal, error)
	Available(ctx context.Context, q domain.AvailabilityQuery) ([]domain.AvailableRoom, error)
	List(ctx context.Context, f domain.BookingFilter) ([]domain.Booking, error)
	Cancel(ctx context.Context, id int64) error
}

type BookingService struct {
	repo bookingRepository
	loc  *time.Location
}

func NewBookingService(repo bookingRepository, loc *time.Location) *BookingService {
	return &BookingService{repo: repo, loc: loc}
}

func (s *BookingService) Create(ctx context.Context, in domain.CreateBookingInput) (int64, decimal.Decimal, error) {
	if in.RoomID <= 0 {
		return 0, decimal.Zero, invalid("invalid room id")
	}

	in.GuestName = strings.TrimSpace(in.GuestName)
	if in.GuestName == "" {
		return 0, decimal.Zero, invalid("guest_name cannot be empty")
	}

	in.GuestEmail = strings.ToLower(strings.TrimSpace(in.GuestEmail))
	if in.GuestEmail == "" {
		return 0, decimal.Zero, invalid("guest_email cannot be empty")
	}

	if in.GuestPhone != nil {
		phone := strings.TrimSpace(*in.GuestPhone)
		if phone == "" {
			in.GuestPhone = nil
		} else {
			in.GuestPhone = &phone
		}
	}

	if !in.CheckOut.After(in.CheckIn) {
		return 0, decimal.Zero, invalid("check_out must be after check_in")
	}

	if in.CheckIn.Before(s.today()) {
		return 0, decimal.Zero, invalid("check_in cannot be in the past")
	}

	return s.repo.Insert(ctx, in)
}

func (s *BookingService) Available(ctx context.Context, q domain.AvailabilityQuery) ([]domain.AvailableRoom, error) {
	if q.HotelID <= 0 {
		return nil, invalid("invalid hotel id")
	}

	if !q.To.After(q.From) {
		return nil, invalid("to must be after from")
	}

	if q.From.Before(s.today()) {
		return nil, invalid("from cannot be in the past")
	}

	if q.Guests <= 0 {
		return nil, invalid("guests must be a positive integer")
	}

	return s.repo.Available(ctx, q)
}

func (s *BookingService) List(ctx context.Context, f domain.BookingFilter) ([]domain.Booking, error) {
	if f.HotelID != nil && *f.HotelID <= 0 {
		return nil, invalid("invalid hotel_id")
	}

	if f.Status != nil && *f.Status != "confirmed" && *f.Status != "cancelled" {
		return nil, invalid("invalid status")
	}

	return s.repo.List(ctx, f)
}

func (s *BookingService) Cancel(ctx context.Context, id int64) error {
	if id <= 0 {
		return invalid("invalid booking id")
	}

	return s.repo.Cancel(ctx, id)
}

func (s *BookingService) today() time.Time {
	return dateIn(time.Now(), s.loc)
}

func dateIn(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
