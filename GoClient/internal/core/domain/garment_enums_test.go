package domain

import (
	"errors"
	"testing"

	"weicloth/internal/core/apperrors"
)

func TestValidateGarmentStatus(t *testing.T) {
	tests := []struct {
		status string
		ok     bool
	}{
		{"queued", true},
		{"processing", true},
		{"completed", true},
		{"failed", true},
		{"bogus", false},
		{"", false},
	}
	for _, tc := range tests {
		err := ValidateGarmentStatus(tc.status)
		if tc.ok && err != nil {
			t.Errorf("status %q: want nil, got %v", tc.status, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("status %q: want error", tc.status)
		}
		if !tc.ok && !errors.Is(err, apperrors.ErrInvalidInput) {
			t.Errorf("status %q: want ErrInvalidInput, got %v", tc.status, err)
		}
	}
}

func TestValidateGarmentType(t *testing.T) {
	if err := ValidateGarmentType("Shirt"); err != nil {
		t.Fatalf("Shirt should be valid (case-insensitive): %v", err)
	}
	if err := ValidateGarmentType("hat"); err == nil {
		t.Fatal("hat should be invalid")
	}
}

func TestValidateOptionalGarmentStatus(t *testing.T) {
	if err := ValidateOptionalGarmentStatus(""); err != nil {
		t.Fatalf("empty status should be allowed: %v", err)
	}
}
