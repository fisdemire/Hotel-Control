package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type AuthConfig struct {
	JWTSecret string
	TokenTTL  time.Duration
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host,
		p.Port,
		p.User,
		p.Password,
		p.Database,
		p.SSLMode,
	)
}

func Load() (*Config, error) {
	port, err := getInt("DB_PORT", 5432)
	if err != nil {
		return nil, err
	}

	maxConns, err := getInt32("DB_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}

	minConns, err := getInt32("DB_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}

	maxConnLifetime, err := getDuration(
		"DB_MAX_CONN_LIFETIME",
		30*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	maxConnIdleTime, err := getDuration(
		"DB_MAX_CONN_IDLE_TIME",
		5*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	user, err := getRequired("DB_USER")
	if err != nil {
		return nil, err
	}

	password, err := getRequired("DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	database, err := getRequired("DB_NAME")
	if err != nil {
		return nil, err
	}

	authSecret, err := getRequired("AUTH_SECRET")
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Addr:            getString("SERVER_ADDR", ":8080"),
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 15 * time.Second,
		},
		Postgres: PostgresConfig{
			Host:            getString("DB_HOST", "localhost"),
			Port:            port,
			User:            user,
			Password:        password,
			Database:        database,
			SSLMode:         getString("DB_SSLMODE", "disable"),
			MaxConns:        maxConns,
			MinConns:        minConns,
			MaxConnLifetime: maxConnLifetime,
			MaxConnIdleTime: maxConnIdleTime,
		},
		Auth: AuthConfig{
			JWTSecret: authSecret,
			TokenTTL:  time.Hour,
		},
	}, nil
}

func getString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func getRequired(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}

	return value, nil
}

func getInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be integer: %w", key, err)
	}

	return result, nil
}

func getInt32(key string, defaultValue int32) (int32, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	result, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be int32: %w", key, err)
	}

	return int32(result), nil
}

func getDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be duration: %w", key, err)
	}

	return result, nil
}
