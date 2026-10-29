package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// typeNames gives every struct type of one outline output its own name. A
// struct is named after its key (an array's element struct after the key
// and "Item"); when that name is taken, by the root struct or by another
// struct, a number is appended (Meta2). Before, both were declared, and the
// output did not compile.
//
// The structs kirke generated before arrays were typed from all of their
// elements (from an array's first element when it is not null) are named
// first, by their path, so that no output that compiled before changes a
// type name: a struct that only merging creates (from a later element, or
// an array whose first element is null) takes the next free name. A struct
// with the same base name and the same fields as an earlier one that is
// not named first is that earlier type.
type typeNames struct {
	used   map[string]bool
	next   map[string]int    // last number appended to a base name
	defs   map[string]string // base name and fields → type name
	legacy map[string]string // path → name of a struct named first
	path   string            // path of the value being typed
}

func newTypeNames(reserved ...string) *typeNames {
	n := &typeNames{used: map[string]bool{}, next: map[string]int{}, defs: map[string]string{},
		legacy: map[string]string{}}
	for _, name := range reserved {
		n.used[name] = true
	}
	return n
}

// reserve returns base, or base with the next number from 2 appended that
// is not taken, and takes it.
func (n *typeNames) reserve(base string) string {
	name := base
	for n.used[name] {
		if n.next[base] == 0 {
			n.next[base] = 1
		}
		n.next[base]++
		name = fmt.Sprintf("%s%d", base, n.next[base])
	}
	n.used[name] = true
	return name
}

// reserveLegacyFields names the structs under the fields of o (at path)
// that kirke generated from the first element of each array.
func (n *typeNames) reserveLegacyFields(o *OrderedMap, path string) {
	names := uniqueFieldNames(o.Keys)
	for i, k := range o.Keys {
		if _, isNull := o.NullFields[k]; !isNull {
			n.reserveLegacy(o.Map[k], path+"\x00."+k, names[i])
		}
	}
}

func (n *typeNames) reserveLegacy(v interface{}, path, base string) {
	switch v := v.(type) {
	case *OrderedMap:
		n.legacy[path] = n.reserve(base)
		n.reserveLegacyFields(v, path)
	case []interface{}:
		if len(v) > 0 {
			n.reserveLegacy(v[0], path+"\x00[]", base+"Item")
		}
	}
}

func generateOutlineStruct(structName string, data OrderedMap, withPointer bool, nullTypeName string) string {
	names := newTypeNames(structName)
	names.reserveLegacyFields(&data, "")
	fields, nested := outlineFields(names, data, withPointer, nullTypeName)
	return "type " + structName + " struct {\n" + fields + "}\n\n" + nested
}

// outlineFields returns the field declarations of the struct for data and
// the declarations of the structs its fields use.
func outlineFields(names *typeNames, data OrderedMap, withPointer bool, nullTypeName string) (string, string) {
	var sb strings.Builder
	var nestedStructs strings.Builder

	path := names.path
	for i, fieldName := range fieldNames(data.Keys) {
		keyName := data.Keys[i]
		names.path = path + "\x00." + keyName
		fieldType, nestedDef := goTypeForOutline(names, fieldName, keyName, data.Map[keyName],
			withPointer, data.NumberStrings, data.BoolFields, data.NullFields, nullTypeName)
		sb.WriteString(fmt.Sprintf("\t%s %s %s\n", fieldName, fieldType, jsonTag(keyName)))
		nestedStructs.WriteString(nestedDef)
	}
	names.path = path
	return sb.String(), nestedStructs.String()
}

func GoTypeForOutline(fieldName string, keyName string, value interface{}, withPointer bool,
	numberStrings map[string]string, boolFields map[string]bool, nullFields map[string]string, nullTypeName string) (string, string) {
	return goTypeForOutline(newTypeNames(), fieldName, keyName, value, withPointer,
		numberStrings, boolFields, nullFields, nullTypeName)
}

func goTypeForOutline(names *typeNames, fieldName string, keyName string, value interface{}, withPointer bool,
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
			elemType, representative, hasNull, ok := scalarArrayElemType(v, nullTypeName)
			if ok {
				return "[]" + elemType, ""
			}
			path := names.path
			names.path = path + "\x00[]"
			elemType, nestedDef := goTypeForOutline(names, fieldName+"Item", "", representative,
				withPointer, numberStrings, boolFields, nullFields, nullTypeName)
			names.path = path
			return "[]" + nullableElem(elemType, hasNull), nestedDef
		}
		return "[]" + nullTypeName, ""
	case mergedValues:
		elemType, representative, hasNull, ok := scalarArrayElemType(v, nullTypeName)
		if ok {
			return elemType, ""
		}
		fieldType, nestedDef := goTypeForOutline(names, fieldName, keyName, representative,
			withPointer, numberStrings, boolFields, nullFields, nullTypeName)
		return nullableElem(fieldType, hasNull), nestedDef
	case *OrderedMap:
		typeName, named := names.legacy[names.path]
		if !named {
			typeName = names.reserve(fieldName)
		}
		fields, nested := outlineFields(names, *v, withPointer, nullTypeName)
		structDef := "type " + typeName + " struct {\n" + fields + "}\n\n" + nested
		key := fieldName + "\x00" + fields
		if earlier, ok := names.defs[key]; ok && !named {
			delete(names.used, typeName)
			typeName, structDef = earlier, ""
		} else if !ok {
			names.defs[key] = typeName
		}
		if withPointer {
			return "*" + typeName, structDef
		}
		return typeName, structDef

	default:
		return "interface{}", ""
	}
}
