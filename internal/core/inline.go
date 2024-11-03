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
