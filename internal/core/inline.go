package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/magicdrive/kirke/internal/common"
)

func generateInlineStruct(structName string, data OrderedMap) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, keyName := range data.Keys {
		fieldName := common.ToCamelCase(keyName)
		fieldType, nestedFields := GoTypeForInline(fieldName, data.Map[keyName], data.NumberStrings, data.BoolFields)

		if nestedFields != "" {
			sb.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} `json:\"%s\"`\n", fieldName, nestedFields, keyName))
		} else {
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, keyName))
		}
	}

	sb.WriteString("}\n\n")
	return sb.String()
}

func GoTypeForInline(fieldName string, value interface{}, numberStrings map[string]string, boolFields map[string]bool) (string, string) {
	if numStr, exists := numberStrings[common.ToSnakeCase(fieldName)]; exists {
		return parseNumber(json.Number(numStr)), ""
	}

	if _, exists := boolFields[fieldName]; exists {
		return "bool", ""
	}

	switch v := value.(type) {
	case *OrderedMap:
		return "", generateInlineFields(*v)
	case map[string]interface{}:
		nestedMap := OrderedMap{Map: v}
		for k := range v {
			nestedMap.Keys = append(nestedMap.Keys, k)
		}
		return "", generateInlineFields(nestedMap)
	case []interface{}:
		if len(v) > 0 {
			elemType, nestedFields := GoTypeForInline(fieldName+"Item", v[0], numberStrings, boolFields)
			if nestedFields != "" {
				return "[]struct {\n" + nestedFields + "\t}", ""
			}
			return "[]" + elemType, ""
		}
		return "[]interface{}", ""
	default:
		return reflect.TypeOf(value).String(), ""
	}
}

func generateInlineFields(data OrderedMap) string {
	var fields strings.Builder

	for _, key := range data.Keys {
		fieldName := common.ToCamelCase(key)
		fieldType, nestedFields := GoTypeForInline(fieldName, data.Map[key], data.NumberStrings, data.BoolFields)

		if nestedFields != "" {
			fields.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} `json:\"%s\"`\n", fieldName, nestedFields, key))
		} else {
			fields.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, key))
		}
	}

	return fields.String()
}
