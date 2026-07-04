package handlers

import (
	"strings"
	"testing"
)

func TestValidateExperimentFields(t *testing.T) {
	validDescription := strings.Repeat("d", experimentDescriptionMaxLength)
	tooLongDescription := strings.Repeat("d", experimentDescriptionMaxLength+1)

	testCases := []struct {
		name        string
		experiment  string
		key         string
		description *string
		wantErr     bool
	}{
		{name: "valid", experiment: "Experiment", key: "experiment-key", wantErr: false},
		{name: "name too long", experiment: strings.Repeat("a", experimentNameMaxLength+1), key: "experiment-key", wantErr: true},
		{name: "key too long", experiment: "Experiment", key: strings.Repeat("k", experimentKeyMaxLength+1), wantErr: true},
		{name: "description max length", experiment: "Experiment", key: "experiment-key", description: &validDescription, wantErr: false},
		{name: "description too long", experiment: "Experiment", key: "experiment-key", description: &tooLongDescription, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nameErr := validateExperimentName(tc.experiment)
			keyErr := validateExperimentKey(tc.key)
			descriptionErr := validateExperimentDescription(tc.description)
			gotErr := nameErr != nil || keyErr != nil || descriptionErr != nil
			if tc.wantErr != gotErr {
				t.Fatalf("expected error=%v, got nameErr=%v keyErr=%v descriptionErr=%v", tc.wantErr, nameErr, keyErr, descriptionErr)
			}
		})
	}
}
