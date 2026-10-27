package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/magicdrive/kirke/internal/common"
)

func generateInlineStruct(structName string, data OrderedMap, nullTypeName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, keyName := range data.Keys {
		fieldName := common.ToCamelCase(keyName)
		fieldType, nestedFields := GoTypeForInline(fieldName, keyName, data.Map[keyName], data.NumberStrings,
			data.BoolFields, data.NullFields, nullTypeName)

		if nestedFields != "" {
			sb.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} `json:\"%s\"`\n", fieldName, nestedFields, keyName))
		} else {
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, keyName))
		}
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

func GoTypeForInline(fieldName string, keyName string, value interface{}, numberStrings map[string]string,
	boolFields map[string]bool, nullFields map[string]string, nullTypeName string) (string, string) {

	if numStr, exists := numberStrings[keyName]; exists {
		return parseNumber(json.Number(numStr)), ""
	}

	if _, exists := boolFields[keyName]; exists {
		return "bool", ""
	}

	if _, exists := nullFields[keyName]; exists {
		return nullTypeName, ""
	}

	switch v := value.(type) {
	case *OrderedMap:
		return "", generateInlineFields(*v, nullTypeName)
	case map[string]interface{}:
		nestedMap := OrderedMap{Map: v}
		for k := range v {
			nestedMap.Keys = append(nestedMap.Keys, k)
		}
		return "", generateInlineFields(nestedMap, nullTypeName)
	case []interface{}:
		if len(v) > 0 {
			if elemType, ok := overflowArrayElemType(v); ok {
				return "[]" + elemType, ""
			}
			if v[0] == nil {
				// kirke used to panic here (reflect has no type for nil).
				return "[]" + nullableInlineElemType(fieldName, keyName, v, numberStrings,
					boolFields, nullFields, nullTypeName), ""
			}
			if _, ok := v[0].(json.Number); ok {
				// Array numbers used to be decoded as float64; they keep
				// that type.
				return "[]float64", ""
			}
			elemType, nestedFields := GoTypeForInline(fieldName+"Item", keyName,
				v[0], numberStrings, boolFields, nullFields, nullTypeName)
			if nestedFields != "" {
				return "[]struct {\n" + nestedFields + "\t}", ""
			}
			return "[]" + elemType, ""
		}
		return "[]" + nullTypeName, ""
	default:
		return reflect.TypeOf(value).String(), ""
	}
}

// nullableInlineElemType returns the element type of an array whose first
// element is null: interface{} when every element is (as in outline
// mode), else a pointer to the type of the first element that is not null
// (*float64 for a number), so null decodes to nil and encodes back to null;
// an array element stays a slice, which null already decodes to nil.
func nullableInlineElemType(fieldName string, keyName string, values []interface{}, numberStrings map[string]string,
	boolFields map[string]bool, nullFields map[string]string, nullTypeName string) string {
	first := firstNonNull(values)
	if first == nil {
		return "interface{}"
	}
	if _, ok := first.(json.Number); ok {
		return "*float64"
	}
	elemType, nestedFields := GoTypeForInline(fieldName+"Item", keyName,
		first, numberStrings, boolFields, nullFields, nullTypeName)
	switch {
	case nestedFields != "":
		return "*struct {\n" + nestedFields + "\t}"
	case strings.HasPrefix(elemType, "[]"):
		return elemType
	default:
		return "*" + elemType
	}
}

func generateInlineFields(data OrderedMap, nullValue string) string {
	var fields strings.Builder

	for _, keyName := range data.Keys {
		fieldName := common.ToCamelCase(keyName)
		fieldType, nestedFields := GoTypeForInline(fieldName, keyName, data.Map[keyName], data.NumberStrings,
			data.BoolFields, data.NullFields, nullValue)

		if nestedFields != "" {
			fields.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} `json:\"%s\"`\n", fieldName, nestedFields, keyName))
		} else {
			fields.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, keyName))
		}
	}

	return fields.String()
}
