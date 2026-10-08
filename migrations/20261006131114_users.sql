-- +goose Up

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    email TEXT NOT NULL UNIQUE
        CHECK (email = lower(email)),

    password_hash TEXT NOT NULL,

    name TEXT NOT NULL,

    phone TEXT,

    role TEXT NOT NULL
        CHECK (role IN ('guest', 'manager', 'admin')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE users;