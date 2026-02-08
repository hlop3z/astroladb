package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// UserBase is used for creating and updating user
type UserBase struct {
	Age        *int64           `json:"age"`
	Avatar     *string          `json:"avatar"`
	Balance    decimal.Decimal  `json:"balance"`
	Bio        *string          `json:"bio"`
	Birthdate  *time.Time       `json:"birthdate"`
	ExternalId *uuid.UUID       `json:"external_id"`
	IsActive   bool             `json:"is_active"`
	LastSeen   *time.Time       `json:"last_seen"`
	LoginTime  *time.Time       `json:"login_time"`
	Role       interface{}      `json:"role"`
	Score      *float64         `json:"score"`
	Settings   *json.RawMessage `json:"settings"`
	Username   string           `json:"username"`
}

// User represents a user record
type User struct {
	ID         uuid.UUID        `json:"id"`
	Age        *int64           `json:"age"`
	Avatar     *string          `json:"avatar"`
	Balance    decimal.Decimal  `json:"balance"`
	Bio        *string          `json:"bio"`
	Birthdate  *time.Time       `json:"birthdate"`
	ExternalId *uuid.UUID       `json:"external_id"`
	IsActive   bool             `json:"is_active"`
	LastSeen   *time.Time       `json:"last_seen"`
	LoginTime  *time.Time       `json:"login_time"`
	Role       interface{}      `json:"role"`
	Score      *float64         `json:"score"`
	Settings   *json.RawMessage `json:"settings"`
	Username   string           `json:"username"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}
