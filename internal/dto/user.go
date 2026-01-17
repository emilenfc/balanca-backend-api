package dto

import "github.com/google/uuid"

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Balance     int64     `json:"balance"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   string    `json:"created_at"`
}

type UpdateUserRequest struct {
	Email     string `json:"email" binding:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type UserSearchResponse struct {
	ID          uuid.UUID `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
}