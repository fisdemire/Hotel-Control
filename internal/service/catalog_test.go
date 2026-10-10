package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/shopspring/decimal"
)

type fakeCatalogRepo struct {
	catalogRepository
	hotel *domain.CreateHotelInput
}

func (f *fakeCatalogRepo) InsertHotel(_ context.Context, in domain.CreateHotelInput) (domain.Hotel, error) {
	f.hotel = &in

	return domain.Hotel{ID: 1, Name: in.Name, Address: in.Address}, nil
}

func TestCreateHotel_TrimsInput(t *testing.T) {
	repo := &fakeCatalogRepo{}
	svc := NewCatalogService(repo)

	_, err := svc.CreateHotel(context.Background(), domain.CreateHotelInput{
		Name:    "  Grand  ",
		Address: " Main st 1 ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.hotel.Name != "Grand" || repo.hotel.Address != "Main st 1" {
		t.Fatalf("input was not trimmed: %+v", *repo.hotel)
	}
}

func TestCreateHotel_RejectsEmpty(t *testing.T) {
	svc := NewCatalogService(&fakeCatalogRepo{})

	_, err := svc.CreateHotel(context.Background(), domain.CreateHotelInput{Name: "  ", Address: "x"})

	var inputErr *domain.InputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("want domain.InputError, got %v", err)
	}
}

func TestCreateRoomType_Validation(t *testing.T) {
	svc := NewCatalogService(&fakeCatalogRepo{})

	tests := []struct {
		name    string
		hotelID int64
		in      domain.CreateRoomTypeInput
	}{
		{"bad hotel id", 0, domain.CreateRoomTypeInput{Name: "std", Capacity: 2}},
		{"empty name", 1, domain.CreateRoomTypeInput{Name: " ", Capacity: 2}},
		{"zero capacity", 1, domain.CreateRoomTypeInput{Name: "std", Capacity: 0}},
		{"negative price", 1, domain.CreateRoomTypeInput{Name: "std", Capacity: 2, PricePerNight: decimal.NewFromInt(-1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateRoomType(context.Background(), tt.hotelID, tt.in)

			var inputErr *domain.InputError
			if !errors.As(err, &inputErr) {
				t.Fatalf("want domain.InputError, got %v", err)
			}
		})
	}
}

func TestUpdateRoomStatus_RejectsUnknownStatus(t *testing.T) {
	svc := NewCatalogService(&fakeCatalogRepo{})

	_, err := svc.UpdateRoomStatus(context.Background(), 1, "broken")

	var inputErr *domain.InputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("want domain.InputError, got %v", err)
	}
}
