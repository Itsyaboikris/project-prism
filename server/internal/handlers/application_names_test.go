package handlers

import (
	"strings"
	"testing"
)

func TestValidateApplicationName(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty", value: "", wantErr: true},
		{name: "valid", value: "Project Prism", wantErr: false},
		{name: "max length", value: strings.Repeat("a", applicationNameMaxLength), wantErr: false},
		{name: "too long", value: strings.Repeat("a", applicationNameMaxLength+1), wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateApplicationName(tc.value)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
