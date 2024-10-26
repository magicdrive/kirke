package core_test

import (
	"encoding/json"
	"go/format"
	"reflect"
	"testing"

	"strings"

	"github.com/magicdrive/kirke/internal/core"
)

func TestApply(t *testing.T) {
	tests := []struct {
		jsonStr     string
		rootObjName string
		withPointer bool
		expected    string
		expectError bool
	}{
		{
			jsonStr:     `{"name": "Alice", "age": 30, "is_member": true}`,
			rootObjName: "User",
			withPointer: false,
			expected:    "type User struct {\n\tName string `json:\"name\"`\n\tAge int `json:\"age\"`\n\tIsMember bool `json:\"is_member\"`\n}\n",
			expectError: false,
		},
		{
			jsonStr:     `{"person": {"name": "Alice"}, "is_active": true}`,
			rootObjName: "Response",
			withPointer: true,
			expected:    "type Response struct {\n\tPerson *Person `json:\"person\"`\n\tIsActive bool `json:\"is_active\"`\n}\n\ntype Person struct {\n\tName string `json:\"name\"`\n}\n",
			expectError: false,
		},
	}

	for _, tt := range tests {
		// フォーマット済みの期待結果を生成
		formattedExpected, err := format.Source([]byte(tt.expected))
		if err != nil {
			t.Fatalf("Error formatting expected source: %v", err)
		}

		// Apply関数の実行
		result, err := core.Apply(tt.jsonStr, tt.rootObjName, tt.withPointer)
		if (err != nil) != tt.expectError {
			t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
		}

		// 空白と改行を除去して比較
		expectedStr := strings.ReplaceAll(string(formattedExpected), "\n", "")
		expectedStr = strings.ReplaceAll(expectedStr, "\t", "")
		expectedStr = strings.ReplaceAll(expectedStr, " ", "")

		resultStr := strings.ReplaceAll(result, "\n", "")
		resultStr = strings.ReplaceAll(resultStr, "\t", "")
		resultStr = strings.ReplaceAll(resultStr, " ", "")

		if expectedStr != resultStr {
			t.Errorf("Expected:\n%s\nGot:\n%s", string(formattedExpected), result)
		}
	}
}

func TestOrderedMap_UnmarshalJSON(t *testing.T) {
	jsonStr := `{"name": "Alice", "age": 30}`
	var om core.OrderedMap
	err := om.UnmarshalJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedKeys := []string{"name", "age"}
	if !reflect.DeepEqual(om.Keys, expectedKeys) {
		t.Errorf("Expected keys: %v, got: %v", expectedKeys, om.Keys)
	}

	expectedMap := map[string]interface{}{
		"name": "Alice",
		"age":  json.Number("30"),
	}
	if !reflect.DeepEqual(om.Map, expectedMap) {
		t.Errorf("Expected map: %v, got: %v", expectedMap, om.Map)
	}
}

func TestGoType(t *testing.T) {
	tests := []struct {
		fieldName    string
		value        interface{}
		withPointer  bool
		expectedType string
	}{
		{"FieldString", "sample", false, "string"},
		{"FieldNumber", json.Number("123"), false, "int"},
		{"FieldBool", true, false, "bool"},
		{"FieldSlice", []interface{}{"item1"}, false, "[]string"},
		{"FieldMap", map[string]interface{}{"nested": "value"}, false, "FieldMap"},
	}

	for _, tt := range tests {
		gotType, _ := core.GoType(tt.fieldName, tt.value, tt.withPointer)
		if gotType != tt.expectedType {
			t.Errorf("For fieldName %s, expected type %s, got %s", tt.fieldName, tt.expectedType, gotType)
		}
	}
}
