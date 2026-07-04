package handlers

import "fmt"

const applicationNameMaxLength = 80

func validateApplicationName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > applicationNameMaxLength {
		return fmt.Errorf("name must be 80 characters or fewer")
	}

	return nil
}
