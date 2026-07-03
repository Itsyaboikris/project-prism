package models

import "time"

type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusActive    ExperimentStatus = "active"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
)

func (s ExperimentStatus) Valid() bool {
	switch s {
	case ExperimentStatusDraft, ExperimentStatusActive, ExperimentStatusPaused, ExperimentStatusCompleted:
		return true
	}
	return false
}

type Experiment struct {
	ID            string           `json:"id"`
	ApplicationID string           `json:"application_id"`
	Key           string           `json:"key"`
	Name          string           `json:"name"`
	Description   *string          `json:"description"`
	Status        ExperimentStatus `json:"status"`
	StartDate     *time.Time       `json:"start_date"`
	EndDate       *time.Time       `json:"end_date"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Branches      []*Branch        `json:"branches"`
}
