package repository

import (
	"context"
	"errors"

	"github.com/fisdemire/Hotel-Control/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type BookingRepository struct{ db *pgxpool.Pool }

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) Insert(ctx context.Context, in domain.CreateBookingInput) (int64, decimal.Decimal, error) {
	var id int64
	var total decimal.Decimal

	err := r.db.QueryRow(ctx,
		`INSERT INTO bookings (
			room_id,
			guest_name,
			guest_email,
			guest_phone,
			check_in,
			check_out,
			status,
			price_per_night,
			total_price
		)
		SELECT
			r.id,
			$2::text,
			$3::text,
			$4::text,
			$5::date,
			$6::date,
			'confirmed',
			rt.price_per_night,
			rt.price_per_night * ($6::date - $5::date)
		FROM rooms r
		JOIN room_types rt
			ON rt.id = r.room_type_id
		WHERE r.id = $1
		  AND r.status = 'active'
		  AND rt.active
		RETURNING id, total_price`,
		in.RoomID,
		in.GuestName,
		in.GuestEmail,
		in.GuestPhone,
		in.CheckIn,
		in.CheckOut,
	).Scan(&id, &total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, decimal.Zero, domain.ErrRoomUnavailable
		}
		if pgCode(err) == pgExclusionViolation {
			return 0, decimal.Zero, domain.ErrAlreadyBooked
		}
		return 0, decimal.Zero, err
	}

	return id, total, nil
}

func (r *BookingRepository) Available(ctx context.Context, q domain.AvailabilityQuery) ([]domain.AvailableRoom, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			r.id,
			r.number,
			rt.name AS type_name,
			rt.capacity,
			rt.price_per_night,
			rt.price_per_night * ($3::date - $2::date) AS total_price
		FROM rooms r
		JOIN room_types rt
			ON rt.id = r.room_type_id
		WHERE r.hotel_id = $1
		  AND r.status = 'active'
		  AND rt.active
		  AND rt.capacity >= $4
		  AND NOT EXISTS (
			  SELECT 1
			  FROM bookings b
			  WHERE b.room_id = r.id
			    AND b.status = 'confirmed'
			    AND daterange(
				    b.check_in,
				    b.check_out,
				    '[)'
			    ) && daterange(
				    $2::date,
				    $3::date,
				    '[)'
			    )
		  )
		ORDER BY rt.price_per_night, r.number`,
		q.HotelID,
		q.From,
		q.To,
		q.Guests,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.AvailableRoom, 0)

	for rows.Next() {
		var room domain.AvailableRoom

		if err := rows.Scan(
			&room.RoomID,
			&room.Number,
			&room.TypeName,
			&room.Capacity,
			&room.PricePerNight,
			&room.TotalPrice,
		); err != nil {
			return nil, err
		}

		result = append(result, room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *BookingRepository) List(ctx context.Context, f domain.BookingFilter) ([]domain.Booking, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			b.id,
			b.room_id,
			r.number,
			r.hotel_id,
			b.guest_name,
			b.guest_email,
			b.guest_phone,
			b.check_in::text,
			b.check_out::text,
			b.status,
			b.price_per_night,
			b.total_price,
			b.created_at
		FROM bookings b
		JOIN rooms r
			ON r.id = b.room_id
		WHERE ($1::bigint IS NULL OR r.hotel_id = $1)
		  AND ($2::text IS NULL OR b.status = $2)
		  AND ($3::date IS NULL OR b.check_out > $3)
		  AND ($4::date IS NULL OR b.check_in < $4)
		ORDER BY b.check_in DESC
		LIMIT 100`,
		f.HotelID,
		f.Status,
		f.From,
		f.To,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.Booking, 0)

	for rows.Next() {
		var b domain.Booking

		if err := rows.Scan(
			&b.ID,
			&b.RoomID,
			&b.RoomNumber,
			&b.HotelID,
			&b.GuestName,
			&b.GuestEmail,
			&b.GuestPhone,
			&b.CheckIn,
			&b.CheckOut,
			&b.Status,
			&b.PricePerNight,
			&b.TotalPrice,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *BookingRepository) Cancel(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE bookings
		 SET status = 'cancelled'
		 WHERE id = $1
		   AND status = 'confirmed'`,
		id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}

	return nil
}
