package store

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"project-prism/server/internal/models"
)

func TestSelectBranchIsDeterministic(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 20},
		{ID: "branch_variant", Key: "variant-a", Weight: 80},
	}

	first, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != nil {
		t.Fatalf("first selectBranch: %v", err)
	}

	second, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != nil {
		t.Fatalf("second selectBranch: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected deterministic branch, got %q then %q", first.ID, second.ID)
	}
}

func TestSelectBranchHonorsWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 20},
		{ID: "branch_variant", Key: "variant-a", Weight: 80},
	}

	controlUser := findUserIDForBucketRange(t, "app_123", "checkout-button-color", 0, 1999)
	variantUser := findUserIDForBucketRange(t, "app_123", "checkout-button-color", 2000, 9999)

	controlBranch, err := selectBranch("app_123", "checkout-button-color", controlUser, branches)
	if err != nil {
		t.Fatalf("select control branch: %v", err)
	}
	if controlBranch.Key != "control" {
		t.Fatalf("expected control branch, got %q", controlBranch.Key)
	}

	variantBranch, err := selectBranch("app_123", "checkout-button-color", variantUser, branches)
	if err != nil {
		t.Fatalf("select variant branch: %v", err)
	}
	if variantBranch.Key != "variant-a" {
		t.Fatalf("expected variant branch, got %q", variantBranch.Key)
	}
}

func TestSelectBranchRejectsMisconfiguredWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_a", Key: "a", Weight: 0},
		{ID: "branch_b", Key: "b", Weight: 0},
	}

	_, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != ErrMisconfigured {
		t.Fatalf("expected ErrMisconfigured, got %v", err)
	}
}

func findUserIDForBucketRange(t *testing.T, applicationID, experimentKey string, min, max uint64) string {
	t.Helper()

	for i := range 100_000 {
		userID := fmt.Sprintf("user_%d", i)
		bucket := assignmentBucket(applicationID, experimentKey, userID)
		if bucket >= min && bucket <= max {
			return userID
		}
	}

	t.Fatalf("no user id found for bucket range %d-%d", min, max)
	return ""
}

func assignmentBucket(applicationID, experimentKey, userID string) uint64 {
	sum := sha256.Sum256([]byte(applicationID + ":" + experimentKey + ":" + userID))
	return binary.BigEndian.Uint64(sum[:8]) % 10000
}
