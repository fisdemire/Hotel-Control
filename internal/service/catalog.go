package service

import (
	"context"
	"strings"

	"github.com/fisdemire/Hotel-Control/internal/domain"
)

type catalogRepository interface {
	ListHotels(ctx context.Context) ([]domain.Hotel, error)
	InsertHotel(ctx context.Context, in domain.CreateHotelInput) (domain.Hotel, error)

	ListRoomTypes(ctx context.Context, hotelID int64) ([]domain.RoomType, error)
	InsertRoomType(ctx context.Context, hotelID int64, in domain.CreateRoomTypeInput) (domain.RoomType, error)
	UpdateRoomType(ctx context.Context, id int64, in domain.UpdateRoomTypeInput) (domain.RoomType, error)

	ListRooms(ctx context.Context, hotelID int64) ([]domain.RoomView, error)
	InsertRoom(ctx context.Context, hotelID int64, in domain.CreateRoomInput) (domain.Room, error)
	UpdateRoomStatus(ctx context.Context, id int64, status string) (domain.Room, error)
}

type CatalogService struct {
	repo catalogRepository
}

func NewCatalogService(repo catalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) ListHotels(ctx context.Context) ([]domain.Hotel, error) {
	return s.repo.ListHotels(ctx)
}

func (s *CatalogService) CreateHotel(ctx context.Context, in domain.CreateHotelInput) (domain.Hotel, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Address = strings.TrimSpace(in.Address)

	if in.Name == "" {
		return domain.Hotel{}, invalid("name cannot be empty")
	}
	if in.Address == "" {
		return domain.Hotel{}, invalid("address cannot be empty")
	}

	return s.repo.InsertHotel(ctx, in)
}

func (s *CatalogService) ListRoomTypes(ctx context.Context, hotelID int64) ([]domain.RoomType, error) {
	if hotelID <= 0 {
		return nil, invalid("invalid hotel id")
	}

	return s.repo.ListRoomTypes(ctx, hotelID)
}

func (s *CatalogService) CreateRoomType(ctx context.Context, hotelID int64, in domain.CreateRoomTypeInput) (domain.RoomType, error) {
	if hotelID <= 0 {
		return domain.RoomType{}, invalid("invalid hotel id")
	}

	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return domain.RoomType{}, invalid("name cannot be empty")
	}
	if in.Capacity <= 0 {
		return domain.RoomType{}, invalid("capacity must be greater than zero")
	}
	if in.PricePerNight.IsNegative() {
		return domain.RoomType{}, invalid("price_per_night must be non-negative")
	}

	return s.repo.InsertRoomType(ctx, hotelID, in)
}

func (s *CatalogService) UpdateRoomType(ctx context.Context, id int64, in domain.UpdateRoomTypeInput) (domain.RoomType, error) {
	if id <= 0 {
		return domain.RoomType{}, invalid("invalid room type id")
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return domain.RoomType{}, invalid("name cannot be empty")
		}
		in.Name = &name
	}
	if in.Capacity != nil && *in.Capacity <= 0 {
		return domain.RoomType{}, invalid("capacity must be greater than zero")
	}
	if in.PricePerNight != nil && in.PricePerNight.IsNegative() {
		return domain.RoomType{}, invalid("price_per_night must be non-negative")
	}

	return s.repo.UpdateRoomType(ctx, id, in)
}

func (s *CatalogService) ListRooms(ctx context.Context, hotelID int64) ([]domain.RoomView, error) {
	if hotelID <= 0 {
		return nil, invalid("invalid hotel id")
	}

	return s.repo.ListRooms(ctx, hotelID)
}

func (s *CatalogService) CreateRoom(ctx context.Context, hotelID int64, in domain.CreateRoomInput) (domain.Room, error) {
	if hotelID <= 0 {
		return domain.Room{}, invalid("invalid hotel id")
	}
	if in.RoomTypeID <= 0 {
		return domain.Room{}, invalid("invalid room type id")
	}

	in.Number = strings.TrimSpace(in.Number)
	if in.Number == "" {
		return domain.Room{}, invalid("number cannot be empty")
	}

	return s.repo.InsertRoom(ctx, hotelID, in)
}

func (s *CatalogService) UpdateRoomStatus(ctx context.Context, id int64, status string) (domain.Room, error) {
	if id <= 0 {
		return domain.Room{}, invalid("invalid room id")
	}

	switch status {
	case domain.StatusActive, domain.StatusMaintenance, domain.StatusDisabled:
	default:
		return domain.Room{}, invalid("invalid status")
	}

	return s.repo.UpdateRoomStatus(ctx, id, status)
}
