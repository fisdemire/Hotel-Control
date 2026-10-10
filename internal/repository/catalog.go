package repository

import (
	"context"
	"errors"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	hotelCols    = `id, name, address, created_at`
	roomTypeCols = `id, hotel_id, name, capacity, price_per_night, active`
	roomCols     = `id, hotel_id, room_type_id, number, status, created_at`
)

type CatalogRepository struct{ db *pgxpool.Pool }

func NewCatalogRepository(db *pgxpool.Pool) *CatalogRepository { return &CatalogRepository{db: db} }

func (r *CatalogRepository) ListHotels(ctx context.Context) ([]domain.Hotel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+hotelCols+` FROM hotels ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Hotel, error) {
		return scanHotel(row)
	})
}

func (r *CatalogRepository) InsertHotel(ctx context.Context, in domain.CreateHotelInput) (domain.Hotel, error) {
	return scanHotel(r.db.QueryRow(ctx,
		`INSERT INTO hotels (name, address)
		 VALUES ($1, $2)
		 RETURNING `+hotelCols,
		in.Name,
		in.Address,
	))
}

func (r *CatalogRepository) ListRoomTypes(ctx context.Context, hotelID int64) ([]domain.RoomType, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+roomTypeCols+`
		 FROM room_types
		 WHERE hotel_id = $1
		 ORDER BY id`,
		hotelID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.RoomType, error) {
		return scanRoomType(row)
	})
}

func (r *CatalogRepository) InsertRoomType(ctx context.Context, hotelID int64, in domain.CreateRoomTypeInput) (domain.RoomType, error) {

	rt, err := scanRoomType(r.db.QueryRow(ctx,
		`INSERT INTO room_types (hotel_id, name, capacity, price_per_night)
		 SELECT id, $2::text, $3::int, $4::numeric
		 FROM hotels
		 WHERE id = $1
		 RETURNING `+roomTypeCols,
		hotelID,
		in.Name,
		in.Capacity,
		in.PricePerNight,
	))
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return domain.RoomType{}, domain.ErrHotelNotFound
		case pgCode(err) == pgUniqueViolation:
			return domain.RoomType{}, domain.ErrRoomTypeExists
		}
		return domain.RoomType{}, err
	}

	return rt, nil
}

func (r *CatalogRepository) UpdateRoomType(ctx context.Context, id int64, in domain.UpdateRoomTypeInput) (domain.RoomType, error) {
	rt, err := scanRoomType(r.db.QueryRow(ctx,
		`UPDATE room_types
		 SET name            = COALESCE($2, name),
		     capacity        = COALESCE($3, capacity),
		     price_per_night = COALESCE($4, price_per_night),
		     active          = COALESCE($5, active)
		 WHERE id = $1
		 RETURNING `+roomTypeCols,
		id,
		in.Name,
		in.Capacity,
		in.PricePerNight,
		in.Active,
	))
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return domain.RoomType{}, domain.ErrRoomTypeNotFound
		case pgCode(err) == pgUniqueViolation:
			return domain.RoomType{}, domain.ErrRoomTypeExists
		}
		return domain.RoomType{}, err
	}

	return rt, nil
}

func (r *CatalogRepository) ListRooms(ctx context.Context, hotelID int64) ([]domain.RoomView, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			r.id,
			r.hotel_id,
			r.room_type_id,
			r.number,
			r.status,
			r.created_at,
			rt.name,
			rt.price_per_night
		 FROM rooms r
		 JOIN room_types rt ON rt.id = r.room_type_id
		 WHERE r.hotel_id = $1
		 ORDER BY r.number`,
		hotelID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.RoomView, error) {
		var v domain.RoomView

		err := row.Scan(
			&v.ID,
			&v.HotelID,
			&v.RoomTypeID,
			&v.Number,
			&v.Status,
			&v.CreatedAt,
			&v.TypeName,
			&v.PricePerNight,
		)

		return v, err
	})
}

func (r *CatalogRepository) InsertRoom(ctx context.Context, hotelID int64, in domain.CreateRoomInput) (domain.Room, error) {
	room, err := scanRoom(r.db.QueryRow(ctx,
		`INSERT INTO rooms (hotel_id, room_type_id, number, status)
		 VALUES ($1, $2, $3, 'active')
		 RETURNING `+roomCols,
		hotelID,
		in.RoomTypeID,
		in.Number,
	))
	if err != nil {
		switch pgCode(err) {
		case pgUniqueViolation:
			return domain.Room{}, domain.ErrRoomNumberExists
		case pgForeignKeyViolation:
			return domain.Room{}, domain.ErrInvalidReference
		}
		return domain.Room{}, err
	}

	return room, nil
}

func (r *CatalogRepository) UpdateRoomStatus(ctx context.Context, id int64, status string) (domain.Room, error) {
	room, err := scanRoom(r.db.QueryRow(ctx,
		`UPDATE rooms
		 SET status = $2
		 WHERE id = $1
		 RETURNING `+roomCols,
		id,
		status,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Room{}, domain.ErrRoomNotFound
		}
		return domain.Room{}, err
	}

	return room, nil
}

func scanHotel(row pgx.Row) (domain.Hotel, error) {
	var h domain.Hotel
	err := row.Scan(&h.ID, &h.Name, &h.Address, &h.CreatedAt)

	return h, err
}

func scanRoomType(row pgx.Row) (domain.RoomType, error) {
	var rt domain.RoomType
	err := row.Scan(&rt.ID, &rt.HotelID, &rt.Name, &rt.Capacity, &rt.PricePerNight, &rt.Active)

	return rt, err
}

func scanRoom(row pgx.Row) (domain.Room, error) {
	var r domain.Room
	err := row.Scan(&r.ID, &r.HotelID, &r.RoomTypeID, &r.Number, &r.Status, &r.CreatedAt)

	return r, err
}
