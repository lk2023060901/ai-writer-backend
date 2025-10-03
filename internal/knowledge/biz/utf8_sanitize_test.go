package biz

import (
	"testing"
	"unicode/utf8"
)

// TestSanitizeUTF8 tests the UTF-8 sanitization function that fixes PostgreSQL encoding errors
func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		shouldValid bool
		description string
	}{
		{
			name:        "ValidUTF8",
			input:       []byte("Hello, World! This is valid UTF-8."),
			shouldValid: true,
			description: "Plain ASCII should remain unchanged",
		},
		{
			name:        "InvalidByte0xBA",
			input:       []byte{'H', 'e', 'l', 'l', 'o', 0xBA, 'W', 'o', 'r', 'l', 'd'},
			shouldValid: false, // Input is invalid
			description: "Contains 0xBA which caused PostgreSQL UTF-8 error",
		},
		{
			name:        "MultipleInvalidBytes",
			input:       []byte{'T', 'e', 's', 't', 0xBA, 0xBB, 'E', 'n', 'd'},
			shouldValid: false,
			description: "Multiple invalid bytes should all be replaced",
		},
		{
			name:        "EmptyString",
			input:       []byte{},
			shouldValid: true,
			description: "Empty input should produce empty valid output",
		},
		{
			name:        "OnlyInvalidBytes",
			input:       []byte{0xBA, 0xBB, 0xBC},
			shouldValid: false,
			description: "All invalid bytes should be replaced with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := string(tt.input)

			// Verify input validation status matches expectation
			if utf8.ValidString(input) != tt.shouldValid {
				t.Errorf("Input validity mismatch: expected %v, got %v", tt.shouldValid, utf8.ValidString(input))
			}

			// Sanitize the input
			result := sanitizeUTF8(input)

			// Result should always be valid UTF-8
			if !utf8.ValidString(result) {
				t.Errorf("Result is not valid UTF-8: %q", result)
			}

			// If input was valid, result should be identical
			if tt.shouldValid && result != input {
				t.Errorf("Valid input was changed: input=%q, result=%q", input, result)
			}

			// If input was invalid, result should be different and contain spaces
			if !tt.shouldValid && result == input {
				t.Errorf("Invalid input was not sanitized: %q", input)
			}

			t.Logf("%s: input_valid=%v, input=%q, output=%q",
				tt.description, tt.shouldValid, input, result)
		})
	}
}

// TestSanitizeUTF8_RealWorldCase tests the actual error case from production
func TestSanitizeUTF8_RealWorldCase(t *testing.T) {
	// This is the actual byte sequence that caused:
	// ERROR: invalid byte sequence for encoding "UTF8": 0xba (SQLSTATE 22021)
	invalidText := []byte{
		'S', 'o', 'm', 'e', ' ', 't', 'e', 'x', 't', ' ',
		0xBA, // Invalid UTF-8 byte that caused PostgreSQL error
		' ', 'c', 'o', 'n', 't', 'i', 'n', 'u', 'e', 's',
	}

	input := string(invalidText)

	// Verify this input is indeed invalid UTF-8
	if utf8.ValidString(input) {
		t.Fatalf("Test case setup error: input should be invalid UTF-8")
	}

	// Sanitize
	result := sanitizeUTF8(input)

	// Result must be valid UTF-8
	if !utf8.ValidString(result) {
		t.Errorf("Sanitized result is not valid UTF-8")
	}

	// Result should contain the valid parts
	if len(result) == 0 {
		t.Errorf("Sanitized result is empty, expected non-empty string")
	}

	// The invalid byte should be replaced
	if result == input {
		t.Errorf("Result unchanged, invalid byte was not replaced")
	}

	t.Logf("Original (invalid): %q", input)
	t.Logf("Sanitized (valid):  %q", result)
}

// TestSanitizeUTF8_PreservesValidContent tests that valid content is preserved
func TestSanitizeUTF8_PreservesValidContent(t *testing.T) {
	validInputs := []string{
		"Hello, World!",
		"Test123",
		"email@example.com",
		"Line 1\nLine 2\nLine 3",
		"Tab\tseparated\tvalues",
		"Special chars: !@#$%^&*()",
	}

	for _, input := range validInputs {
		t.Run(input, func(t *testing.T) {
			result := sanitizeUTF8(input)

			if result != input {
				t.Errorf("Valid input was modified: input=%q, output=%q", input, result)
			}

			if !utf8.ValidString(result) {
				t.Errorf("Result is not valid UTF-8: %q", result)
			}
		})
	}
}
