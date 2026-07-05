package models

import "time"

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
)

func (r UserRole) Valid() bool {
	switch r {
	case UserRoleAdmin:
		return true
	}

	return false
}

type UserStatus string

const (
	UserStatusInvited  UserStatus = "invited"
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

func (s UserStatus) Valid() bool {
	switch s {
	case UserStatusInvited, UserStatusActive, UserStatusInactive:
		return true
	}

	return false
}

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash *string    `json:"-"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

type RefreshToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"-"`
}

type InvitationToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	ConsumedAt *time.Time `json:"-"`
}
