package rooms

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

func Register(
	r *gin.Engine,
	db *pgxpool.Pool,
	adminMiddleware gin.HandlerFunc,
	staffMiddleware gin.HandlerFunc,
) {
	r.GET(
		"/hotels",
		getHotel(db),
	)

	r.POST(
		"/hotels",
		adminMiddleware,
		createHotel(db),
	)

	r.GET(
		"/hotels/:id/room-types",
		listRoomTypes(db),
	)

	r.POST(
		"/hotels/:id/room-types",
		adminMiddleware,
		createRoomType(db),
	)

	r.PATCH(
		"/room-types/:id",
		adminMiddleware,
		updateRoomType(db),
	)

	r.GET(
		"/hotels/:id/rooms",
		listRooms(db),
	)

	r.POST(
		"/hotels/:id/rooms",
		adminMiddleware,
		staffMiddleware,
		createRoom(db),
	)

	r.PATCH(
		"/rooms/:id/status",
		adminMiddleware,
		staffMiddleware,
		updateRoomStatus(db),
	)
}

func getHotel(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(
			c.Request.Context(),
			`SELECT
				id,
				name,
				address,
				created_at
			FROM hotels
			ORDER BY id`,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
		defer rows.Close()

		type hotelResponse struct {
			ID        int64     `json:"id"`
			Name      string    `json:"name"`
			Address   string    `json:"address"`
			CreatedAt time.Time `json:"created_at"`
		}

		hotels := make([]hotelResponse, 0)

		for rows.Next() {
			var hotel hotelResponse

			if err := rows.Scan(
				&hotel.ID,
				&hotel.Name,
				&hotel.Address,
				&hotel.CreatedAt,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				return
			}

			hotels = append(hotels, hotel)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, hotels)
	}
}

func createHotel(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string `json:"name" binding:"required"`
			Address string `json:"address" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		name := strings.TrimSpace(req.Name)
		address := strings.TrimSpace(req.Address)

		if name == "" || address == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		var hotel struct {
			ID        int64     `json:"id"`
			Name      string    `json:"name"`
			Address   string    `json:"address"`
			CreatedAt time.Time `json:"created_at"`
		}

		err := db.QueryRow(
			c.Request.Context(),
			`INSERT INTO hotels (
					name,
					address
				)
				VALUES ($1, $2)
				RETURNING
					id,
					name,
					address,
					created_at`,
			name,
			address,
		).Scan(
			&hotel.ID,
			&hotel.Name,
			&hotel.Address,
			&hotel.CreatedAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusCreated, hotel)
	}
}

func listRoomTypes(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || hotelID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid hotel id",
			})
			return
		}

		rows, err := db.Query(
			c.Request.Context(),
			`SELECT
				id,
				hotel_id,
				name,
				capacity,
				price_per_night,
				active
			FROM room_types
			WHERE hotel_id = $1
			ORDER BY id`,
			hotelID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
		defer rows.Close()

		type roomTypeResponse struct {
			ID            int64           `json:"id"`
			HotelID       int64           `json:"hotel_id"`
			Name          string          `json:"name"`
			Capacity      int             `json:"capacity"`
			PricePerNight decimal.Decimal `json:"price_per_night"`
			Active        bool            `json:"active"`
		}

		roomTypes := make([]roomTypeResponse, 0)

		for rows.Next() {
			var roomType roomTypeResponse

			if err := rows.Scan(
				&roomType.ID,
				&roomType.HotelID,
				&roomType.Name,
				&roomType.Capacity,
				&roomType.PricePerNight,
				&roomType.Active,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				return
			}

			roomTypes = append(roomTypes, roomType)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, roomTypes)
	}
}

func createRoomType(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || hotelID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid hotel id",
			})
			return
		}

		var req struct {
			Name          string          `json:"name" binding:"required"`
			Capacity      int             `json:"capacity" binding:"required"`
			PricePerNight decimal.Decimal `json:"price_per_night" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		name := strings.TrimSpace(req.Name)

		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		if req.Capacity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "capacity must be greater than zero",
			})
			return
		}

		if req.PricePerNight.IsNegative() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "price_per_night must be non-negative",
			})
			return
		}

		var roomType struct {
			ID            int64           `json:"id"`
			HotelID       int64           `json:"hotel_id"`
			Name          string          `json:"name"`
			Capacity      int             `json:"capacity"`
			PricePerNight decimal.Decimal `json:"price_per_night"`
			Active        bool            `json:"active"`
		}

		err = db.QueryRow(
			c.Request.Context(),
			`INSERT INTO room_types (
					hotel_id,
					name,
					capacity,
					price_per_night
				)
				SELECT
					id,
					$2,
					$3,
					$4
				FROM hotels
				WHERE id = $1
				RETURNING
					id,
					hotel_id,
					name,
					capacity,
					price_per_night,
					active`,
			hotelID,
			name,
			req.Capacity,
			req.PricePerNight,
		).Scan(
			&roomType.ID,
			&roomType.HotelID,
			&roomType.Name,
			&roomType.Capacity,
			&roomType.PricePerNight,
			&roomType.Active,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "hotel not found",
				})
				return
			}

			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{
					"error": "room type already exists",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusCreated, roomType)
	}
}

func updateRoomType(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomTypeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || roomTypeID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid room type id",
			})
			return
		}

		var req struct {
			Name          *string          `json:"name"`
			Capacity      *int             `json:"capacity"`
			PricePerNight *decimal.Decimal `json:"price_per_night"`
			Active        *bool            `json:"active"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		if req.Name != nil {
			*req.Name = strings.TrimSpace(*req.Name)

			if *req.Name == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "name cannot be empty",
				})
				return
			}
		}

		if req.Capacity != nil && *req.Capacity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "capacity must be greater than zero",
			})
			return
		}

		if req.PricePerNight != nil && req.PricePerNight.IsNegative() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "price_per_night must be non-negative",
			})
			return
		}

		var roomType struct {
			ID            int64           `json:"id"`
			HotelID       int64           `json:"hotel_id"`
			Name          string          `json:"name"`
			Capacity      int             `json:"capacity"`
			PricePerNight decimal.Decimal `json:"price_per_night"`
			Active        bool            `json:"active"`
		}

		err = db.QueryRow(
			c.Request.Context(),
			`UPDATE room_types
				SET
					name = COALESCE($2, name),
					capacity = COALESCE($3, capacity),
					price_per_night = COALESCE($4, price_per_night),
					active = COALESCE($5, active)
				WHERE id = $1
				RETURNING
					id,
					hotel_id,
					name,
					capacity,
					price_per_night,
					active`,
			roomTypeID,
			req.Name,
			req.Capacity,
			req.PricePerNight,
			req.Active,
		).Scan(
			&roomType.ID,
			&roomType.HotelID,
			&roomType.Name,
			&roomType.Capacity,
			&roomType.PricePerNight,
			&roomType.Active,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "room type not found",
				})
				return
			}

			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{
					"error": "room type already exists",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, roomType)
	}
}

func listRooms(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || hotelID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid hotel id",
			})
			return
		}

		rows, err := db.Query(
			c.Request.Context(),
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}
		defer rows.Close()

		type roomResponse struct {
			ID            int64           `json:"id"`
			HotelID       int64           `json:"hotel_id"`
			RoomTypeID    int64           `json:"room_type_id"`
			Number        string          `json:"number"`
			Status        string          `json:"status"`
			CreatedAt     time.Time       `json:"created_at"`
			TypeName      string          `json:"type_name"`
			PricePerNight decimal.Decimal `json:"price_per_night"`
		}

		rooms := make([]roomResponse, 0)

		for rows.Next() {
			var room roomResponse

			if err := rows.Scan(
				&room.ID,
				&room.HotelID,
				&room.RoomTypeID,
				&room.Number,
				&room.Status,
				&room.CreatedAt,
				&room.TypeName,
				&room.PricePerNight,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				return
			}

			rooms = append(rooms, room)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, rooms)
	}
}

func createRoom(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || hotelID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid hotel id",
			})
			return
		}

		var req struct {
			RoomTypeID int64  `json:"room_type_id" binding:"required"`
			Number     string `json:"number" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		req.Number = strings.TrimSpace(req.Number)

		if req.Number == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "number cannot be empty",
			})
			return
		}

		var room struct {
			ID         int64     `json:"id"`
			HotelID    int64     `json:"hotel_id"`
			RoomTypeID int64     `json:"room_type_id"`
			Number     string    `json:"number"`
			Status     string    `json:"status"`
			CreatedAt  time.Time `json:"created_at"`
		}

		err = db.QueryRow(
			c.Request.Context(),
			`INSERT INTO rooms (
				hotel_id,
				room_type_id,
				number,
				status
			)
			VALUES ($1, $2, $3, 'active')
			RETURNING
				id,
				hotel_id,
				room_type_id,
				number,
				status,
				created_at`,
			hotelID,
			req.RoomTypeID,
			req.Number,
		).Scan(
			&room.ID,
			&room.HotelID,
			&room.RoomTypeID,
			&room.Number,
			&room.Status,
			&room.CreatedAt,
		)

		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) {
				switch pgErr.Code {
				case "23505":
					c.JSON(http.StatusConflict, gin.H{
						"error": "room number already exists",
					})
					return

				case "23503":
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "invalid room type or hotel",
					})
					return
				}
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusCreated, room)
	}
}

func updateRoomStatus(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || roomID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid room id",
			})
			return
		}

		var req struct {
			Status string `json:"status" binding:"required,oneof=active maintenance disabled"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad request",
			})
			return
		}

		var room struct {
			ID         int64     `json:"id"`
			HotelID    int64     `json:"hotel_id"`
			RoomTypeID int64     `json:"room_type_id"`
			Number     string    `json:"number"`
			Status     string    `json:"status"`
			CreatedAt  time.Time `json:"created_at"`
		}

		err = db.QueryRow(
			c.Request.Context(),
			`UPDATE rooms
			 SET status = $2
			 WHERE id = $1
			 RETURNING
				id,
				hotel_id,
				room_type_id,
				number,
				status,
				created_at`,
			roomID,
			req.Status,
		).Scan(
			&room.ID,
			&room.HotelID,
			&room.RoomTypeID,
			&room.Number,
			&room.Status,
			&room.CreatedAt,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "room not found",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		c.JSON(http.StatusOK, room)
	}
}
