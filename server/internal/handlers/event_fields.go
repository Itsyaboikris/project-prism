package handlers

import (
	"encoding/json"
	"fmt"
)

const (
	eventNameMaxLength       = 64
	eventPropertiesMaxBytes  = 4096
)

func validateEventName(name string) error {
	if name == "" {
		return fmt.Errorf("event_name is required")
	}
	if len(name) > eventNameMaxLength {
		return fmt.Errorf("event_name must be 64 characters or fewer")
	}
	return nil
}

func validateEventPropertiesJSON(properties json.RawMessage) error {
	if len(properties) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(properties, &value); err != nil {
		return fmt.Errorf("properties must be valid JSON")
	}
	if value == nil {
		return nil
	}

	objectValue, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("properties must be a JSON object")
	}

	serialized, err := json.Marshal(objectValue)
	if err != nil {
		return fmt.Errorf("properties must be valid JSON")
	}
	if len(serialized) > eventPropertiesMaxBytes {
		return fmt.Errorf("properties must be 4096 bytes or fewer")
	}

	return nil
}
