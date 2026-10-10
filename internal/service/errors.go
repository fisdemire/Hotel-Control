package service

import "github.com/fisdemire/Hotel-Control/internal/domain"

func invalid(msg string) error { return domain.NewInputError(msg) }
