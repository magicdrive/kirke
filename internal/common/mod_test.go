package common_test

import (
	"testing"

	"github.com/magicdrive/kirke/internal/common"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello_world", "HelloWorld"},
		{"test_case_function", "TestCaseFunction"},
		{"AlreadyCamelCase", "AlreadyCamelCase"},
		{"simple_test", "SimpleTest"},
		{"", ""},
		{"multiple__underscores", "MultipleUnderscores"},
		{"with_numbers_123", "WithNumbers123"},
	}

	for _, tt := range tests {
		result := common.ToCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("For input %q, expected %q, but got %q", tt.input, tt.expected, result)
		}
	}
}

