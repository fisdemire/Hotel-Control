package domain

type User struct {
	ID           int64   `json:"id"`
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	Phone        *string `json:"phone"`
	Role         string  `json:"role"`
	PasswordHash string  `json:"-"`
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Phone    *string
}

type Principal struct {
	UserID int64
	Role   string
}
