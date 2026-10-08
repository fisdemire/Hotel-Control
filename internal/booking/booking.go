package booking

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

const dateLayout = "2006-01-02"

type availabilityResponse struct {
	RoomID        int64           `json:"room_id"`
	Number        string          `json:"number"`
	TypeName      string          `json:"type_name"`
	Capacity      int             `json:"capacity"`
	PricePerNight decimal.Decimal `json:"price_per_night"`
	TotalPrice    decimal.Decimal `json:"total_price"`
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

type bookingResponse struct {
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

func Register(
	r *gin.Engine,
	db *pgxpool.Pool,
	staffMiddleware gin.HandlerFunc,
	adminMiddleware gin.HandlerFunc,
) {
	r.GET(
		"/hotels/:id/availability",
		availability(db),
	)

	r.POST(
		"/bookings",
		createBooking(db),
	)

	r.GET(
		"/bookings",
		staffMiddleware,
		listBookings(db),
	)

	r.POST(
		"/bookings/:id/cancel",
		staffMiddleware,
		cancelBooking(db),
	)
}

func availability(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID, err := strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)
		if err != nil || hotelID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid hotel id",
			})
			return
		}

		from, err := parseDate(c.Query("from"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid from date",
			})
			return
		}

		to, err := parseDate(c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid to date",
			})
			return
		}

		if !to.After(from) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "to must be after from",
			})
			return
		}

		today := time.Now().UTC().Truncate(24 * time.Hour)

		if from.Before(today) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "from cannot be in the past",
			})
			return
		}

		guests := 1

		if guestsParam := c.Query("guests"); guestsParam != "" {
			guests, err = strconv.Atoi(guestsParam)
			if err != nil || guests <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "guests must be a positive integer",
				})
				return
			}
		}

		rows, err := db.Query(
			c.Request.Context(),
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
			hotelID,
			from,
			to,
			guests,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
		defer rows.Close()

		result := make([]availabilityResponse, 0)

		for rows.Next() {
			var room availabilityResponse

			if err := rows.Scan(
				&room.RoomID,
				&room.Number,
				&room.TypeName,
				&room.Capacity,
				&room.PricePerNight,
				&room.TotalPrice,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				return
			}

			result = append(result, room)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func createBooking(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createBookingRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		if req.RoomID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid room id",
			})
			return
		}

		guestName := strings.TrimSpace(req.GuestName)

		if guestName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "guest_name cannot be empty",
			})
			return
		}

		guestEmail := strings.ToLower(
			strings.TrimSpace(req.GuestEmail),
		)

		if guestEmail == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "guest_email cannot be empty",
			})
			return
		}

		var guestPhone any

		if req.GuestPhone != nil {
			phone := strings.TrimSpace(*req.GuestPhone)

			if phone != "" {
				guestPhone = phone
			}
		}

		checkIn, err := parseDate(req.CheckIn)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid check_in date",
			})
			return
		}

		checkOut, err := parseDate(req.CheckOut)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid check_out date",
			})
			return
		}

		if !checkOut.After(checkIn) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "check_out must be after check_in",
			})
			return
		}

		today := time.Now().UTC().Truncate(24 * time.Hour)

		if checkIn.Before(today) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "check_in cannot be in the past",
			})
			return
		}

		var response createBookingResponse

		err = db.QueryRow(
			c.Request.Context(),
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
			req.RoomID,
			guestName,
			guestEmail,
			guestPhone,
			checkIn,
			checkOut,
		).Scan(
			&response.ID,
			&response.TotalPrice,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "room not found or unavailable",
				})
				return
			}

			if pgCode(err) == "23P01" {
				c.JSON(http.StatusConflict, gin.H{
					"error": "room is already booked for these dates",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusCreated, response)
	}
}

func listBookings(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hotelID *int64

		if value := c.Query("hotel_id"); value != "" {
			parsed, err := strconv.ParseInt(
				value,
				10,
				64,
			)
			if err != nil || parsed <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid hotel_id",
				})
				return
			}

			hotelID = &parsed
		}

		var status *string

		if value := c.Query("status"); value != "" {
			if value != "confirmed" && value != "cancelled" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid status",
				})
				return
			}

			status = &value
		}

		var from *time.Time

		if value := c.Query("from"); value != "" {
			parsed, err := parseDate(value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from date",
				})
				return
			}

			from = &parsed
		}

		var to *time.Time

		if value := c.Query("to"); value != "" {
			parsed, err := parseDate(value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid to date",
				})
				return
			}

			to = &parsed
		}

		rows, err := db.Query(
			c.Request.Context(),
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
			hotelID,
			status,
			from,
			to,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
		defer rows.Close()

		result := make([]bookingResponse, 0)

		for rows.Next() {
			var booking bookingResponse

			if err := rows.Scan(
				&booking.ID,
				&booking.RoomID,
				&booking.RoomNumber,
				&booking.HotelID,
				&booking.GuestName,
				&booking.GuestEmail,
				&booking.GuestPhone,
				&booking.CheckIn,
				&booking.CheckOut,
				&booking.Status,
				&booking.PricePerNight,
				&booking.TotalPrice,
				&booking.CreatedAt,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				return
			}

			result = append(result, booking)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func cancelBooking(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		bookingID, err := strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)
		if err != nil || bookingID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid booking id",
			})
			return
		}

		result, err := db.Exec(
			c.Request.Context(),
			`UPDATE bookings
			 SET status = 'cancelled'
			 WHERE id = $1
			   AND status = 'confirmed'`,
			bookingID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		if result.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "booking not found or already cancelled",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":     bookingID,
			"status": "cancelled",
		})
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(
		dateLayout,
		value,
		time.UTC,
	)
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	return ""
}
