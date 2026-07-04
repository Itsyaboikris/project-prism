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
}

type ExperimentDashboard struct {
	ExperimentID     string                       `json:"experiment_id"`
	ExperimentKey    string                       `json:"experiment_key"`
	ExperimentName   string                       `json:"experiment_name"`
	ExperimentStatus ExperimentStatus             `json:"experiment_status"`
	TotalAssignments int                          `json:"total_assignments"`
	BranchCount      int                          `json:"branch_count"`
	Branches         []*ExperimentDashboardBranch `json:"branches"`
}
