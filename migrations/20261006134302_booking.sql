-- +goose Up

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE bookings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    room_id BIGINT NOT NULL
        REFERENCES rooms(id),

    guest_name TEXT NOT NULL,
    guest_email TEXT NOT NULL,
    guest_phone TEXT,

    check_in DATE NOT NULL,
    check_out DATE NOT NULL,

    status TEXT NOT NULL
        CHECK (
            status IN (
                'confirmed',
                'cancelled'
            )
        ),

    price_per_night NUMERIC(12, 2) NOT NULL
        CHECK (price_per_night >= 0),

    total_price NUMERIC(12, 2) NOT NULL
        CHECK (total_price >= 0),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (check_out > check_in),

    CHECK (
        total_price =
        price_per_night * (check_out - check_in)
    )
);

ALTER TABLE bookings
ADD CONSTRAINT no_overlapping_bookings
EXCLUDE USING gist (
    room_id WITH =,
    daterange(check_in, check_out, '[)') WITH &&
)
WHERE (status = 'confirmed');

CREATE INDEX bookings_check_in_idx
    ON bookings (check_in);

-- +goose Down

DROP TABLE bookings;