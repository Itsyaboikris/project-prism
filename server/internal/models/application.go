package models

import "time"

type ApplicationStatus string

const (
	ApplicationStatusActive   ApplicationStatus = "active"
	ApplicationStatusInactive ApplicationStatus = "inactive"
)

func (s ApplicationStatus) Valid() bool {
	switch s {
	case ApplicationStatusActive, ApplicationStatusInactive:
		return true
	}
	return false
}

type Application struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	APIKey    string            `json:"api_key"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	DeletedAt *time.Time        `json:"-"`
}
