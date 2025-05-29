package model

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"` // Do not expose password hash in JSON responses
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}