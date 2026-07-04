package handlers

import (
	"fmt"
	"math"

	"project-prism/server/internal/models"
)

const (
	branchWeightFractionTotal = 1.0
	branchWeightPercentTotal  = 100.0
	branchWeightTolerance     = 0.0001
)

func validateBranchWeightValue(weight float64) error {
	if weight < 0 || weight > branchWeightPercentTotal {
		return fmt.Errorf("branch weights must be between 0 and 100")
	}
	return nil
}

func validateBranchWeights(branches []*models.Branch) error {
	if len(branches) == 0 {
		return nil
	}

	total := 0.0
	allFractional := true

	for _, branch := range branches {
		if err := validateBranchWeightValue(branch.Weight); err != nil {
			return err
		}
		total += branch.Weight
		if branch.Weight > branchWeightFractionTotal+branchWeightTolerance {
			allFractional = false
		}
	}

	if nearlyEqual(total, branchWeightPercentTotal) {
		return nil
	}
	if allFractional && nearlyEqual(total, branchWeightFractionTotal) {
		return nil
	}

	return fmt.Errorf("branch weights must add up to 100")
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= branchWeightTolerance
}
