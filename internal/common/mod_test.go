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

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CamelCaseString", "camel_case_string"},
		{"anotherExample", "another_example"},
		{"Test123String", "test123_string"},
		{"already_snake_case", "already_snake_case"},
		{"Singleword", "singleword"},
		{"snake_case_with_numbers123", "snake_case_with_numbers123"},
		{"ComplexCaseWithABC123", "complex_case_with_abc123"},
		{"camelCaseWith1Number", "camel_case_with1_number"},
	}

	for _, tt := range tests {
		result := common.ToSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("ToSnakeCase(%s): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}
