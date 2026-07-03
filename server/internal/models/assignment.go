package models

import (
	"encoding/json"
	"time"
)

type Assignment struct {
	ID            string          `json:"id"`
	ApplicationID string          `json:"application_id"`
	ExperimentID  string          `json:"experiment_id"`
	BranchID      string          `json:"branch_id"`
	UserID        string          `json:"user_id"`
	AssignedAt    time.Time       `json:"assigned_at"`
	ContextJSON   json.RawMessage `json:"context_json"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
