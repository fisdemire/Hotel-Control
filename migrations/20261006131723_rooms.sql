-- +goose Up

CREATE TABLE hotels (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    name TEXT NOT NULL,
    address TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE room_types (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    hotel_id BIGINT NOT NULL
        REFERENCES hotels(id),

    name TEXT NOT NULL,

    capacity INT NOT NULL
        CHECK (capacity > 0),

    price_per_night NUMERIC(12, 2) NOT NULL
        CHECK (price_per_night >= 0),

    active BOOLEAN NOT NULL DEFAULT TRUE,

    UNIQUE (hotel_id, name),

    UNIQUE (id, hotel_id)
);

CREATE TABLE rooms (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    hotel_id BIGINT NOT NULL
        REFERENCES hotels(id),

    room_type_id BIGINT NOT NULL,

    number TEXT NOT NULL,

    status TEXT NOT NULL
        CHECK (
            status IN (
                'active',
                'maintenance',
                'disabled'
            )
        ),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (hotel_id, number),

    FOREIGN KEY (room_type_id, hotel_id)
        REFERENCES room_types (id, hotel_id)
);

CREATE INDEX rooms_room_type_id_idx ON rooms (room_type_id);

-- +goose Down

DROP TABLE rooms;
DROP TABLE room_types;
DROP TABLE hotels;