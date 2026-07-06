package models

import "time"

type TrackedEvent struct {
	ID             string     `json:"id"`
	ExperimentID   string     `json:"experiment_id"`
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	Description    *string    `json:"description"`
	OccurrenceCount int       `json:"occurrence_count"`
	LastOccurredAt *time.Time `json:"last_occurred_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}
