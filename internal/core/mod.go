package core

import (
	"encoding/json"
	"fmt"
	"go/format"
	"strings"

	"github.com/magicdrive/kirke/internal/common"
)

func Apply(jsonStr string, rootObjName string, withPointer bool) (string, error) {

	var data OrderedMap
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}

	structDef := generateStruct(common.ToCamelCase(rootObjName), data, withPointer)

	formattedCode, err := format.Source([]byte(structDef))
	if err != nil {
		return "", fmt.Errorf("formatting source: %w \n\nSOURCE:\n%s", err, structDef)
	}

	return string(formattedCode), nil
}

func generateStruct(structName string, data OrderedMap, withPointer bool) string {
	var sb strings.Builder
	var nestedStructs strings.Builder

	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, key := range data.Keys {
		fieldName := common.ToCamelCase(key)
		fieldType, nestedDef := GoType(fieldName, data.Map[key], withPointer, data.NumberStrings, data.BoolFields)
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, key))
		nestedStructs.WriteString(nestedDef)
	}

	sb.WriteString("}\n\n")
	sb.WriteString(nestedStructs.String())
	return sb.String()
}

func GoType(fieldName string, value interface{}, withPointer bool, numberStrings map[string]string, boolFields map[string]bool) (string, string) {

	if numStr, exists := numberStrings[common.ToSnakeCase(fieldName)]; exists {
		return parseNumber(json.Number(numStr)), ""
	}

	if _, exists := boolFields[fieldName]; exists {
		return "bool", ""
	}

	switch v := value.(type) {
	case string:
		return "string", ""
	case bool:
		return "bool", ""
	case []interface{}:
		if len(v) > 0 {
			elemType, nestedDef := GoType(fieldName+"Item", v[0], withPointer, numberStrings, boolFields)
			return "[]" + elemType, nestedDef
		}
		return "[]interface{}", ""
	case *OrderedMap:
		structDef := generateStruct(fieldName, *v, withPointer)
		if withPointer {
			return "*" + fieldName, structDef
		}
		return fieldName, structDef

	default:
		return "interface{}", ""
	}
}
