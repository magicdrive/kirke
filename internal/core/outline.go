package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/magicdrive/kirke/internal/common"
)

func generateOutlineStruct(structName string, data OrderedMap, withPointer bool, nullTypeName string) string {
	var sb strings.Builder
	var nestedStructs strings.Builder

	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, keyName := range data.Keys {
		fieldName := common.ToCamelCase(keyName)
		fieldType, nestedDef := GoTypeForOutline(fieldName, keyName, data.Map[keyName],
			withPointer, data.NumberStrings, data.BoolFields, data.NullFields, nullTypeName)
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, keyName))
		nestedStructs.WriteString(nestedDef)
	}

	sb.WriteString("}\n\n")
	sb.WriteString(nestedStructs.String())
	return sb.String()
}

func GoTypeForOutline(fieldName string, keyName string, value interface{}, withPointer bool,
	numberStrings map[string]string, boolFields map[string]bool, nullFields map[string]string, nullTypeName string) (string, string) {
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
	case string:
		return "string", ""
	case bool:
		return "bool", ""
	case []interface{}:
		if len(v) > 0 {
			elemType, nestedDef := GoTypeForOutline(fieldName+"Item", "", v[0],
				withPointer, numberStrings, boolFields, nullFields, nullTypeName)
			return "[]" + elemType, nestedDef
		}
		return "[]interface{}", ""
	case *OrderedMap:
		structDef := generateOutlineStruct(fieldName, *v, withPointer, nullTypeName)
		if withPointer {
			return "*" + fieldName, structDef
		}
		return fieldName, structDef

	default:
		return "interface{}", ""
	}
}
