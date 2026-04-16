package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserStatus represents lifecycle state for a user.
type UserStatus string

const (
	// UserStatusActive marks an active user.
	UserStatusActive UserStatus = "active"
	// UserStatusInactive marks an inactive user.
	UserStatusInactive UserStatus = "inactive"
)

// User stores wallet owner data.
type User struct {
	BaseModel
	ExternalRef *string        `gorm:"size:100;uniqueIndex" json:"external_ref,omitempty"`
	FullName    string         `gorm:"size:255;not null" json:"full_name"`
	Email       string         `gorm:"size:255;not null;uniqueIndex" json:"email"`
	PhoneNumber string         `gorm:"size:32;not null;uniqueIndex" json:"phone_number"`
	Status      UserStatus     `gorm:"size:16;not null;default:active" json:"status"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate guarantees UUID assignment.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if err := u.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if u.Status == "" {
		u.Status = UserStatusActive
	}

	return nil
}

// UserReference provides a compact user projection.
type UserReference struct {
	ID uuid.UUID `json:"id"`
}
