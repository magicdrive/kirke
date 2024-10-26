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
		{"hello_world", "HelloWorld"},        // 基本ケース
		{"test_case_function", "TestCaseFunction"}, // 複数のアンダースコア
		{"AlreadyCamelCase", "AlreadyCamelCase"}, // 既にCamelCase
		{"simple_test", "SimpleTest"},        // 全小文字
		{"", ""},                             // 空文字
		{"multiple__underscores", "MultipleUnderscores"}, // アンダースコアが連続するケース
		{"with_numbers_123", "WithNumbers123"}, // 数字を含むケース
	}

	for _, tt := range tests {
		result := common.ToCamelCase(tt.input)
		if result != tt.expected {
			t.Errorf("For input %q, expected %q, but got %q", tt.input, tt.expected, result)
		}
	}
}

