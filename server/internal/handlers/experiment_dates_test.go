package handlers

import (
	"testing"
	"time"
)

func TestValidateExperimentDates(t *testing.T) {
	start := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	before := start.Add(-2 * time.Hour)

	testCases := []struct {
		name      string
		startDate *time.Time
		endDate   *time.Time
		wantErr   bool
	}{
		{name: "both nil", wantErr: false},
		{name: "only start", startDate: &start, wantErr: false},
		{name: "only end", endDate: &end, wantErr: false},
		{name: "same instant", startDate: &start, endDate: &start, wantErr: false},
		{name: "end after start", startDate: &start, endDate: &end, wantErr: false},
		{name: "end before start", startDate: &start, endDate: &before, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExperimentDates(tc.startDate, tc.endDate)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
