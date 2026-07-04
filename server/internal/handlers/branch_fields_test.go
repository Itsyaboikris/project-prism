package handlers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateBranchFields(t *testing.T) {
	validMetadata := json.RawMessage(`{"color":"#22c55e"}`)
	arrayMetadata := json.RawMessage(`["bad"]`)
	tooLargeMetadata := json.RawMessage(`{"value":"` + strings.Repeat("a", branchMetadataMaxBytes) + `"}`)

	testCases := []struct {
		name     string
		key      string
		branch   string
		metadata json.RawMessage
		wantErr  bool
	}{
		{name: "valid", key: "control", branch: "Control", metadata: validMetadata, wantErr: false},
		{name: "name too long", key: "control", branch: strings.Repeat("a", branchNameMaxLength+1), wantErr: true},
		{name: "key too long", key: strings.Repeat("k", branchKeyMaxLength+1), branch: "Control", wantErr: true},
		{name: "null metadata allowed", key: "control", branch: "Control", metadata: json.RawMessage("null"), wantErr: false},
		{name: "metadata must be object", key: "control", branch: "Control", metadata: arrayMetadata, wantErr: true},
		{name: "metadata too large", key: "control", branch: "Control", metadata: tooLargeMetadata, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nameErr := validateBranchName(tc.branch)
			keyErr := validateBranchKey(tc.key)
			metadataErr := validateBranchMetadataJSON(tc.metadata)
			gotErr := nameErr != nil || keyErr != nil || metadataErr != nil
			if tc.wantErr != gotErr {
				t.Fatalf("expected error=%v, got nameErr=%v keyErr=%v metadataErr=%v", tc.wantErr, nameErr, keyErr, metadataErr)
			}
		})
	}
}
