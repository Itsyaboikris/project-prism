package models

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID            string          `json:"id"`
	ApplicationID string          `json:"application_id,omitempty"`
	ExperimentID  *string         `json:"experiment_id"`
	BranchID      *string         `json:"branch_id"`
	UserID        string          `json:"user_id"`
	EventName     string          `json:"event_name"`
	Properties    json.RawMessage `json:"properties,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	CreatedAt     time.Time       `json:"created_at,omitempty"`
}
