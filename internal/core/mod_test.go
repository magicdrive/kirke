package core_test

import (
	"encoding/json"
	"go/format"
	"reflect"
	"testing"

	"strings"

	"github.com/magicdrive/kirke/internal/common"
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
		formattedExpected, err := format.Source([]byte(tt.expected))
		if err != nil {
			t.Fatalf("Error formatting expected source: %v", err)
		}

		result, err := core.Apply(tt.jsonStr, tt.rootObjName, tt.withPointer)
		if (err != nil) != tt.expectError {
			t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
		}

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

func TestGoType_NoPointer(t *testing.T) {
	tests := []struct {
		fieldName    string
		value        interface{}
		withPointer  bool
		expectedType string
	}{
		{"FieldString", "sample", false, "string"},
		{"FieldNumberInt", json.Number("123"), false, "int"},
		{"FieldNumberFloat", json.Number("1.23"), false, "float64"},
		{"FieldNumberBigInt", json.Number("123567890123567890123567890"), false, "*big.Int"},
		{"FieldNumberBigFloat", json.Number("1.23567e100"), false, "*big.Float"},
		{"FieldBool", true, false, "bool"},
		{"FieldSlice", []interface{}{"item1"}, false, "[]string"},
		{"FieldMap", &core.OrderedMap{Keys: []string{"nested"}, Map: map[string]interface{}{"nested": "value"}}, false, "FieldMap"},
	}

	for _, tt := range tests {
		snakeFieldName := common.ToSnakeCase(tt.fieldName)
		numberStrings := map[string]string{}
		boolFields := map[string]bool{}

		if num, ok := tt.value.(json.Number); ok {
			numberStrings[snakeFieldName] = num.String()
		} else if boolVal, ok := tt.value.(bool); ok {
			boolFields[snakeFieldName] = boolVal
		}

		gotType, _ := core.GoType(tt.fieldName, tt.value, tt.withPointer, numberStrings, boolFields)
		if gotType != tt.expectedType {
			t.Errorf("For fieldName %s, expected type %s, got %s", tt.fieldName, tt.expectedType, gotType)
		}
	}
}

func TestGoType_WithPointer(t *testing.T) {
	tests := []struct {
		fieldName    string
		value        interface{}
		withPointer  bool
		expectedType string
	}{
		{"FieldString", "sample", true, "string"},
		{"FieldNumberInt", json.Number("123"), true, "int"},
		{"FieldNumberFloat", json.Number("1.23"), true, "float64"},
		{"FieldNumberBigInt", json.Number("123567890123567890123567890"), true, "*big.Int"},
		{"FieldNumberBigFloat", json.Number("1.23567e100"), true, "*big.Float"},
		{"FieldBool", true, true, "bool"},
		{"FieldSlice", []interface{}{"item1"}, true, "[]string"},
		{"FieldMap", &core.OrderedMap{Keys: []string{"nested"}, Map: map[string]interface{}{"nested": "value"}}, true, "*FieldMap"},
	}

	for _, tt := range tests {
		snakeFieldName := common.ToSnakeCase(tt.fieldName)
		numberStrings := map[string]string{}
		boolFields := map[string]bool{}

		if num, ok := tt.value.(json.Number); ok {
			numberStrings[snakeFieldName] = num.String()
		} else if boolVal, ok := tt.value.(bool); ok {
			boolFields[snakeFieldName] = boolVal
		}

		gotType, _ := core.GoType(tt.fieldName, tt.value, tt.withPointer, numberStrings, boolFields)
		if gotType != tt.expectedType {
			t.Errorf("For fieldName %s, expected type %s, got %s", tt.fieldName, tt.expectedType, gotType)
		}
	}
}

