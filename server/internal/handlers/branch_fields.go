package handlers

import (
	"encoding/json"
	"fmt"
)

const (
	branchNameMaxLength    = 64
	branchKeyMaxLength     = 64
	branchMetadataMaxBytes = 4096
)

func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name is required")
	}
	if len(name) > branchNameMaxLength {
		return fmt.Errorf("branch name must be 64 characters or fewer")
	}
	return nil
}

func validateBranchKey(key string) error {
	if key == "" {
		return fmt.Errorf("branch key is required")
	}
	if len(key) > branchKeyMaxLength {
		return fmt.Errorf("branch key must be 64 characters or fewer")
	}
	return nil
}

func validateBranchMetadataJSON(metadata json.RawMessage) error {
	if len(metadata) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(metadata, &value); err != nil {
		return fmt.Errorf("branch metadata must be valid JSON")
	}
	if value == nil {
		return nil
	}

	objectValue, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("branch metadata must be a JSON object")
	}

	serialized, err := json.Marshal(objectValue)
	if err != nil {
		return fmt.Errorf("branch metadata must be valid JSON")
	}
	if len(serialized) > branchMetadataMaxBytes {
		return fmt.Errorf("branch metadata must be 4096 bytes or fewer")
	}

	return nil
}
