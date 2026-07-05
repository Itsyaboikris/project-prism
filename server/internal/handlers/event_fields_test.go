package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateEventFields(t *testing.T) {
	validProperties := json.RawMessage(`{"amount":49.99}`)
	arrayProperties := json.RawMessage(`["bad"]`)
	tooLargeProperties := json.RawMessage(`{"value":"` + strings.Repeat("a", eventPropertiesMaxBytes) + `"}`)

	testCases := []struct {
		name       string
		eventName  string
		properties json.RawMessage
		wantErr    bool
	}{
		{name: "valid", eventName: "purchase", properties: validProperties, wantErr: false},
		{name: "missing event name", eventName: "", wantErr: true},
		{name: "event name too long", eventName: strings.Repeat("e", eventNameMaxLength+1), wantErr: true},
		{name: "null properties allowed", eventName: "purchase", properties: json.RawMessage("null"), wantErr: false},
		{name: "properties must be object", eventName: "purchase", properties: arrayProperties, wantErr: true},
		{name: "properties too large", eventName: "purchase", properties: tooLargeProperties, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nameErr := validateEventName(tc.eventName)
			propertiesErr := validateEventPropertiesJSON(tc.properties)
			gotErr := nameErr != nil || propertiesErr != nil
			if tc.wantErr != gotErr {
				t.Fatalf("expected error=%v, got nameErr=%v propertiesErr=%v", tc.wantErr, nameErr, propertiesErr)
			}
		})
	}
}
