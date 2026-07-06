package handlers

import "fmt"

const (
	trackedEventKeyMaxLength         = 64
	trackedEventNameMaxLength        = 64
	trackedEventDescriptionMaxLength = 280
)

func validateTrackedEventKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(key) > trackedEventKeyMaxLength {
		return fmt.Errorf("key must be 64 characters or fewer")
	}
	return nil
}

func validateTrackedEventName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > trackedEventNameMaxLength {
		return fmt.Errorf("name must be 64 characters or fewer")
	}
	return nil
}

func validateTrackedEventDescription(description *string) error {
	if description != nil && len(*description) > trackedEventDescriptionMaxLength {
		return fmt.Errorf("description must be 280 characters or fewer")
	}
	return nil
}
