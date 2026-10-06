include .env

DB_DSN=host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)

migrate-up:
	goose -dir migrations postgres "$(DB_DSN)" up

migrate-down:
	goose -dir migrations postgres "$(DB_DSN)" down

migrate-status:
	goose -dir migrations postgres "$(DB_DSN)" status

migrate-create:
	goose -dir migrations create $(name) sql