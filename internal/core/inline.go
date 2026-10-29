package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func generateInlineStruct(structName string, data OrderedMap, nullTypeName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for i, fieldName := range fieldNames(data.Keys) {
		keyName := data.Keys[i]
		fieldType, nestedFields := GoTypeForInline(fieldName, keyName, data.Map[keyName], data.NumberStrings,
			data.BoolFields, data.NullFields, nullTypeName)

		if nestedFields != "" {
			sb.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} %s\n", fieldName, nestedFields, jsonTag(keyName)))
		} else {
			sb.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag(keyName)))
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
		return inlineStruct(generateInlineFields(*v, nullTypeName))
	case map[string]interface{}:
		nestedMap := OrderedMap{Map: v}
		for k := range v {
			nestedMap.Keys = append(nestedMap.Keys, k)
		}
		return inlineStruct(generateInlineFields(nestedMap, nullTypeName))
	case mergedValues:
		elemType, representative, hasNull, ok := scalarArrayElemType(v, nullTypeName)
		if ok {
			return elemType, ""
		}
		fieldType, nestedFields := GoTypeForInline(fieldName, keyName, representative,
			numberStrings, boolFields, nullFields, nullTypeName)
		if nestedFields != "" && hasNull {
			return "*struct {\n" + nestedFields + "\t}", ""
		}
		return nullableElem(fieldType, hasNull), nestedFields
	case []interface{}:
		if len(v) > 0 {
			elemType, representative, hasNull, ok := scalarArrayElemType(v, nullTypeName)
			if ok {
				return "[]" + elemType, ""
			}
			elemType, nestedFields := GoTypeForInline(fieldName+"Item", keyName,
				representative, numberStrings, boolFields, nullFields, nullTypeName)
			if nestedFields != "" {
				return "[]" + nullableElem("struct {\n"+nestedFields+"\t}", hasNull), ""
			}
			return "[]" + nullableElem(elemType, hasNull), ""
		}
		return "[]" + nullTypeName, ""
	default:
		return reflect.TypeOf(value).String(), ""
	}
}

func generateInlineFields(data OrderedMap, nullValue string) string {
	var fields strings.Builder

	for i, fieldName := range fieldNames(data.Keys) {
		keyName := data.Keys[i]
		fieldType, nestedFields := GoTypeForInline(fieldName, keyName, data.Map[keyName], data.NumberStrings,
			data.BoolFields, data.NullFields, nullValue)

		if nestedFields != "" {
			fields.WriteString(fmt.Sprintf("\t%s struct {\n%s\t} %s\n", fieldName, nestedFields, jsonTag(keyName)))
		} else {
			fields.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag(keyName)))
		}
	}

	return fields.String()
}

// inlineStruct returns the nested fields of an inline struct, or the type
// struct{} for an empty object, which had no type ("X `json:\"x\"`" did
// not compile).
func inlineStruct(fields string) (string, string) {
	if fields == "" {
		return "struct{}", ""
	}
	return "", fields
}
