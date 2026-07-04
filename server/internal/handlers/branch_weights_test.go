package handlers

import (
	"testing"

	"project-prism/server/internal/models"
)

func TestValidateBranchWeights(t *testing.T) {
	testCases := []struct {
		name    string
		weights []float64
		wantErr bool
	}{
		{name: "empty is allowed", weights: nil, wantErr: false},
		{name: "percent total is allowed", weights: []float64{50, 50}, wantErr: false},
		{name: "fractional total is allowed", weights: []float64{0.5, 0.5}, wantErr: false},
		{name: "single branch percent is allowed", weights: []float64{100}, wantErr: false},
		{name: "single branch fractional is allowed", weights: []float64{1}, wantErr: false},
		{name: "negative weight is rejected", weights: []float64{-1, 101}, wantErr: true},
		{name: "weight above 100 is rejected", weights: []float64{120}, wantErr: true},
		{name: "invalid percent total is rejected", weights: []float64{40, 30}, wantErr: true},
		{name: "invalid fractional total is rejected", weights: []float64{0.25, 0.25}, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			branches := make([]*models.Branch, 0, len(tc.weights))
			for _, weight := range tc.weights {
				branches = append(branches, &models.Branch{Weight: weight})
			}

			err := validateBranchWeights(branches)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
