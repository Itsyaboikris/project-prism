package models

import (
	"encoding/json"
	"time"
)

type ExperimentAssignmentListItem struct {
	ID            string          `json:"id"`
	ApplicationID string          `json:"application_id"`
	ExperimentID  string          `json:"experiment_id"`
	BranchID      string          `json:"branch_id"`
	UserID        string          `json:"user_id"`
	AssignedAt    time.Time       `json:"assigned_at"`
	ContextJSON   json.RawMessage `json:"context_json"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	BranchKey     string          `json:"branch_key"`
	BranchName    string          `json:"branch_name"`
	BranchWeight  float64         `json:"branch_weight"`
}

type ExperimentAssignmentsView struct {
	ExperimentID     string                          `json:"experiment_id"`
	ExperimentKey    string                          `json:"experiment_key"`
	ExperimentName   string                          `json:"experiment_name"`
	ExperimentStatus ExperimentStatus                `json:"experiment_status"`
	Assignments      []*ExperimentAssignmentListItem `json:"assignments"`
}

type ExperimentDashboardBranch struct {
	BranchID         string  `json:"branch_id"`
	BranchKey        string  `json:"branch_key"`
	BranchName       string  `json:"branch_name"`
	ConfiguredWeight float64 `json:"configured_weight"`
	AssignmentCount  int     `json:"assignment_count"`
	AssignmentShare  float64 `json:"assignment_share"`
	EventCount       int     `json:"event_count,omitempty"`
	UniqueEventUsers int     `json:"unique_event_users,omitempty"`
	ConversionRate   float64 `json:"conversion_rate,omitempty"`
}

type ExperimentDashboard struct {
	ExperimentID     string                       `json:"experiment_id"`
	ExperimentKey    string                       `json:"experiment_key"`
	ExperimentName   string                       `json:"experiment_name"`
	ExperimentStatus ExperimentStatus             `json:"experiment_status"`
	EventName        string                       `json:"event_name,omitempty"`
	TotalAssignments int                          `json:"total_assignments"`
	BranchCount      int                          `json:"branch_count"`
	Branches         []*ExperimentDashboardBranch `json:"branches"`
}

type ExperimentEventListItem struct {
	ID            string          `json:"id"`
	ApplicationID string          `json:"application_id"`
	ExperimentID  *string         `json:"experiment_id"`
	BranchID      *string         `json:"branch_id"`
	UserID        string          `json:"user_id"`
	EventName     string          `json:"event_name"`
	Properties    json.RawMessage `json:"properties,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	CreatedAt     time.Time       `json:"created_at"`
	BranchKey     *string         `json:"branch_key"`
	BranchName    *string         `json:"branch_name"`
}

type ExperimentEventsView struct {
	ExperimentID     string                   `json:"experiment_id"`
	ExperimentKey    string                   `json:"experiment_key"`
	ExperimentName   string                   `json:"experiment_name"`
	ExperimentStatus ExperimentStatus         `json:"experiment_status"`
	Events           []*ExperimentEventListItem `json:"events"`
}

type EventBranchMetrics struct {
	BranchID         string
	EventCount       int
	UniqueEventUsers int
}
