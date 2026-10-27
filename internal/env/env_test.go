package env

import (
	"os"
	"testing"
)

func TestActive(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true lowercase", "true", true},
		{"true uppercase", "TRUE", true},
		{"true mixed case", "True", true},
		{"false", "false", false},
		{"empty", "", false},
		{"random value", "yes", false},
		{"number", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ACTIVE", tt.val)

			if got := Active("TEST_ACTIVE"); got != tt.want {
				t.Errorf("Active() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("unset env var", func(t *testing.T) {
		os.Unsetenv("TEST_ACTIVE_UNSET")
		if got := Active("TEST_ACTIVE_UNSET"); got {
			t.Errorf("Active() = true, want false for unset env var")
		}
	})
}

func TestEmpty(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"tabs and newlines", "\t\n", true},
		{"non-empty", "value", false},
		{"single space with text", " a ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_EMPTY", tt.val)

			if got := Empty("TEST_EMPTY"); got != tt.want {
				t.Errorf("Empty() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("unset env var", func(t *testing.T) {
		os.Unsetenv("TEST_EMPTY_UNSET")
		if got := Empty("TEST_EMPTY_UNSET"); !got {
			t.Errorf("Empty() = false, want true for unset env var")
		}
	})
}
