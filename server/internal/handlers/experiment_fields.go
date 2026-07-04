package handlers

import "fmt"

const (
	experimentNameMaxLength        = 64
	experimentKeyMaxLength         = 64
	experimentDescriptionMaxLength = 280
)

func validateExperimentName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > experimentNameMaxLength {
		return fmt.Errorf("name must be 64 characters or fewer")
	}
	return nil
}

func validateExperimentKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(key) > experimentKeyMaxLength {
		return fmt.Errorf("key must be 64 characters or fewer")
	}
	return nil
}

func validateExperimentDescription(description *string) error {
	if description != nil && len(*description) > experimentDescriptionMaxLength {
		return fmt.Errorf("description must be 280 characters or fewer")
	}
	return nil
}
