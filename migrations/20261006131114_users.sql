-- +goose Up

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    email TEXT NOT NULL UNIQUE
        CHECK (email = lower(email)),

    password_hash TEXT NOT NULL,

    role TEXT NOT NULL
        CHECK (role IN ('admin', 'manager')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE users;