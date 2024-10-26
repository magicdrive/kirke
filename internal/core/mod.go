package core

import (
	"encoding/json"
	"fmt"
	"go/format"
	"strings"

	"github.com/magicdrive/kirke/internal/common"
)

type OrderedMap struct {
	Keys []string
	Map  map[string]interface{}
}

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

func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	o.Map = make(map[string]interface{})
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return err
	}

	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{', but got %v", tok)
	}

	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}

		switch v := tok.(type) {
		case string:
			o.Keys = append(o.Keys, v)
		default:
			return fmt.Errorf("expected string for key, but got %v", tok)
		}

		var value interface{}
		if err := dec.Decode(&value); err != nil {
			return err
		}
		o.Map[o.Keys[len(o.Keys)-1]] = value
	}

	tok, err = dec.Token()
	if err != nil {
		return err
	}

	if delim, ok := tok.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("expected '}', but got %v", tok)
	}

	return nil
}

func generateStruct(structName string, data OrderedMap, withPointer bool) string {
	var sb strings.Builder
	var nestedStructs strings.Builder

	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, key := range data.Keys {
		fieldName := common.ToCamelCase(key)
		fieldType, newStructs := GoType(fieldName, data.Map[key], withPointer)
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, key))
		nestedStructs.WriteString(newStructs)
	}

	sb.WriteString("}\n\n")
	sb.WriteString(nestedStructs.String())
	return sb.String()
}

func GoType(fieldName string, value interface{}, withPointer bool) (string, string) {

	switch v := value.(type) {
	case string:
		return "string", ""
	case json.Number:
		return parseNumber(v), ""
	case bool:
		return "bool", ""
	case []interface{}:
		if len(v) > 0 {
			elementType, nestedStructs := GoType(fieldName+"Item", v[0], withPointer)
			return "[]" + elementType, nestedStructs
		}
		return "[]interface{}", ""
	case map[string]interface{}:
		nestedData := OrderedMap{Map: v}
		for key := range v {
			nestedData.Keys = append(nestedData.Keys, key)
		}
		nestedStruct := generateStruct(fieldName, nestedData, withPointer)
		if withPointer {
			return "*" + fieldName, nestedStruct
		} else {
			return fieldName, nestedStruct
		}

	default:
		return "interface{}", ""
	}
}
