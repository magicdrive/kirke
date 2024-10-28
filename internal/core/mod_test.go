package core_test

import (
	"encoding/json"
	"go/format"
	"reflect"
	"testing"

	"strings"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/common"
	"github.com/magicdrive/kirke/internal/core"
)

func TestApply(t *testing.T) {
	tests := []struct {
		jsonStr         string
		rootObjName     string
		withPointer     bool
		expectedOutline string
		expectedInline  string
		expectError     bool
	}{
		{
			jsonStr:         `{"name": "Alice", "age": 30, "is_member": true}`,
			rootObjName:     "User",
			withPointer:     false,
			expectedOutline: "type User struct {\n\tName string `json:\"name\"`\n\tAge int `json:\"age\"`\n\tIsMember bool `json:\"is_member\"`\n}\n",
			expectedInline:  "type User struct {\n\tName string `json:\"name\"`\n\tAge int `json:\"age\"`\n\tIsMember bool `json:\"is_member\"`\n}\n",
			expectError:     false,
		},
		{
			jsonStr:         `{"person": {"name": "Alice"}, "is_active": true}`,
			rootObjName:     "Response",
			withPointer:     true,
			expectedOutline: "type Response struct {\n\tPerson *Person `json:\"person\"`\n\tIsActive bool `json:\"is_active\"`\n}\n\ntype Person struct {\n\tName string `json:\"name\"`\n}\n",
			expectedInline:  "type Response struct {\n\tPerson struct {\n\t\tName string `json:\"name\"`\n\t} `json:\"person\"`\n\tIsActive bool `json:\"is_active\"`\n}\n",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		// Test for both OutputModeOutline and OutputModeInline
		modeStrMap := map[int]string{commandline.OutputModeOutline: "Outline", commandline.OutputModeInline: "Inline"}
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			t.Run(tt.rootObjName+"_"+modeStrMap[mode], func(t *testing.T) {
				var expected string
				if mode == commandline.OutputModeOutline {
					expected = tt.expectedOutline
				} else {
					expected = tt.expectedInline
				}

				formattedExpected, err := format.Source([]byte(expected))
				if err != nil {
					t.Fatalf("Error formatting expected source: %v", err)
				}

				result, err := core.Apply(tt.jsonStr, tt.rootObjName, mode, tt.withPointer)
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
			})
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

func TestGoTypeForOutline_NoPointer(t *testing.T) {
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
		{"FieldMap", &core.OrderedMap{Keys: []string{"nested"},
			Map: map[string]interface{}{"nested": "value"}}, false, "FieldMap"},
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

		gotType, _ := core.GoTypeForOutline(tt.fieldName, snakeFieldName, tt.value, tt.withPointer, numberStrings, boolFields)
		if gotType != tt.expectedType {
			t.Errorf("For fieldName %s, expected type %s, got %s", tt.fieldName, tt.expectedType, gotType)
		}
	}
}

func TestGoTypeForOutline_WithPointer(t *testing.T) {
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
		{"FieldMap", &core.OrderedMap{Keys: []string{"nested"},
			Map: map[string]interface{}{"nested": "value"}}, true, "*FieldMap"},
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

		gotType, _ := core.GoTypeForOutline(tt.fieldName, snakeFieldName, tt.value, tt.withPointer, numberStrings, boolFields)
		if gotType != tt.expectedType {
			t.Errorf("For fieldName %s, expected type %s, got %s", tt.fieldName, tt.expectedType, gotType)
		}
	}
}

func TestGoTypeForInline(t *testing.T) {
	tests := []struct {
		fieldName     string
		value         interface{}
		numberStrings map[string]string
		boolFields    map[string]bool
		expectedType  string
		expectedDef   string
	}{
		{
			fieldName:     "FieldNumber",
			value:         json.Number("123"),
			numberStrings: map[string]string{"field_number": "123"},
			expectedType:  "int",
			expectedDef:   "",
		},
		{
			fieldName:    "FieldBool",
			value:        true,
			boolFields:   map[string]bool{"FieldBool": true},
			expectedType: "bool",
			expectedDef:  "",
		},
		{
			fieldName:    "FieldString",
			value:        "sample",
			expectedType: "string",
			expectedDef:  "",
		},
		{
			fieldName: "FieldNestedStruct",
			value: &core.OrderedMap{
				Keys: []string{"id", "name"},
				Map: map[string]interface{}{
					"id":   json.Number("1"),
					"name": "Nested",
				},
			},
			expectedType: "",
			expectedDef:  "\tId json.Number `json:\"id\"`\n\tName string `json:\"name\"`\n",
		},
		{
			fieldName: "FieldArray",
			value: []interface{}{
				map[string]interface{}{"name": "Item1", "id": json.Number("1")},
			},
			expectedType: "[]struct {\n\tName string `json:\"name\"`\n\tId json.Number `json:\"id\"`\n\t}",
			expectedDef:  "",
		},
		{
			fieldName:    "FieldEmptyArray",
			value:        []interface{}{},
			expectedType: "[]interface{}",
			expectedDef:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			fieldType, nestedFields := core.GoTypeForInline(
				tt.fieldName,
				tt.value,
				tt.numberStrings,
				tt.boolFields,
			)

			if fieldType != tt.expectedType {
				t.Errorf("Expected field type: %s, got: %s", tt.expectedType, fieldType)
			}

			if nestedFields != tt.expectedDef {
				t.Errorf("Expected nested fields:\n%s\nGot:\n%s", tt.expectedDef, nestedFields)
			}
		})
	}
}
