package domain

import (
	"strings"

	"weicloth/internal/core/apperrors"
)

var (
	garmentStatuses = map[string]struct{}{
		"queued":     {},
		"processing": {},
		"completed":  {},
		"failed":     {},
	}
	garmentSources = map[string]struct{}{
		"ai":         {},
		"manual":     {},
		"ai+manual":  {},
	}
	garmentTypes = map[string]struct{}{
		"shirt":   {},
		"pants":   {},
		"dress":   {},
		"jacket":  {},
		"shoes":   {},
		"unknown": {},
	}
)

// ValidateGarmentStatus returns an error when status is not a known lifecycle value.
func ValidateGarmentStatus(status string) error {
	if _, ok := garmentStatuses[status]; !ok {
		return &apperrors.ValidationError{Field: "status", Message: "invalid status"}
	}
	return nil
}

// ValidateGarmentSource returns an error when source is not a known origin value.
func ValidateGarmentSource(source string) error {
	if _, ok := garmentSources[source]; !ok {
		return &apperrors.ValidationError{Field: "source", Message: "invalid source"}
	}
	return nil
}

// ValidateGarmentType returns an error when garment_type is not in the product allowlist.
func ValidateGarmentType(garmentType string) error {
	normalized := strings.ToLower(strings.TrimSpace(garmentType))
	if _, ok := garmentTypes[normalized]; !ok {
		return &apperrors.ValidationError{Field: "garment_type", Message: "invalid garment_type"}
	}
	return nil
}

// ValidateOptionalGarmentStatus validates status when non-empty.
func ValidateOptionalGarmentStatus(status string) error {
	if status == "" {
		return nil
	}
	return ValidateGarmentStatus(status)
}

// ValidateOptionalGarmentSource validates source when non-empty.
func ValidateOptionalGarmentSource(source string) error {
	if source == "" {
		return nil
	}
	return ValidateGarmentSource(source)
}

// AllowedGarmentStatuses returns a copy of valid status values (for tests/docs).
func AllowedGarmentStatuses() []string {
	return []string{"queued", "processing", "completed", "failed"}
}

// AllowedGarmentTypes returns a copy of valid garment_type values.
func AllowedGarmentTypes() []string {
	out := make([]string, 0, len(garmentTypes))
	for k := range garmentTypes {
		out = append(out, k)
	}
	return out
}
