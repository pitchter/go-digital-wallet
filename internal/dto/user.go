package dto

import (
	"time"

	"go-digital-wallet/internal/model"

	"github.com/google/uuid"
)

// CreateUserRequest is the POST /users payload.
type CreateUserRequest struct {
	ExternalRef *string          `json:"external_ref,omitempty"`
	FullName    string           `json:"full_name" binding:"required,max=255"`
	Email       string           `json:"email" binding:"required,email,max=255"`
	PhoneNumber string           `json:"phone_number" binding:"required,max=32"`
	Status      model.UserStatus `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
}

// UpdateUserRequest is the PUT /users/:id payload.
type UpdateUserRequest struct {
	ExternalRef *string          `json:"external_ref,omitempty"`
	FullName    string           `json:"full_name" binding:"required,max=255"`
	Email       string           `json:"email" binding:"required,email,max=255"`
	PhoneNumber string           `json:"phone_number" binding:"required,max=32"`
	Status      model.UserStatus `json:"status" binding:"required,oneof=active inactive"`
}

// UserResponse is the user API payload.
type UserResponse struct {
	ID          uuid.UUID        `json:"id"`
	ExternalRef *string          `json:"external_ref,omitempty"`
	FullName    string           `json:"full_name"`
	Email       string           `json:"email"`
	PhoneNumber string           `json:"phone_number"`
	Status      model.UserStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// UserFromModel converts a model into an API payload.
func UserFromModel(user model.User) UserResponse {
	return UserResponse{
		ID:          user.ID,
		ExternalRef: user.ExternalRef,
		FullName:    user.FullName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
