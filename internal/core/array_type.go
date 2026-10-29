package core

import (
	"encoding/json"
	"strings"
)

// scalarArrayElemType returns the element type of a JSON array from all of
// its elements, as decodeJSONArray decodes them (json.Number, string, bool,
// nil, *OrderedMap, []interface{}). ok is false when the elements that are
// not null are all objects or all arrays: the caller then types the array
// from representative, which stands for all of them (mergeElements), and
// makes an object element a pointer when hasNull.
//
//   - no element but null: nullTypeName;
//   - elements of more than one kind (a number and a string, an object and
//     a number): interface{}, or json.RawMessage when one of them is or
//     holds, at any depth, a number float64 does not hold as written
//     (interface{} would decode it into a float64: 1e400 fails, 1e-400
//     becomes 0);
//   - numbers: numberArrayType of all of them;
//   - strings: string; booleans: bool;
//   - a null among numbers, strings or booleans makes the element a
//     pointer (*int, *float64, *json.Number, *string, *bool), so null
//     decodes to nil and encodes back to null; *big.Int already is one.
func scalarArrayElemType(values []interface{}, nullTypeName string) (elemType string, representative interface{}, hasNull bool, ok bool) {
	var nums []json.Number
	kinds := map[string]bool{}
	for _, v := range values {
		switch v := v.(type) {
		case nil:
			hasNull = true
			continue
		case json.Number:
			kinds["number"] = true
			nums = append(nums, v)
		case string:
			kinds["string"] = true
		case bool:
			kinds["bool"] = true
		case *OrderedMap:
			kinds["object"] = true
		case []interface{}:
			kinds["array"] = true
		default:
			kinds["other"] = true
		}
		if representative == nil {
			representative = v
		}
	}
	switch {
	case len(kinds) == 0:
		return nullTypeName, nil, hasNull, true
	case len(kinds) > 1:
		for _, v := range values {
			if !float64Holds(v) {
				return "json.RawMessage", nil, hasNull, true
			}
		}
		return "interface{}", nil, hasNull, true
	case kinds["object"], kinds["array"]:
		return "", mergeElements(values), hasNull, false
	case kinds["other"]:
		return "", representative, hasNull, false
	}
	switch {
	case kinds["number"]:
		elemType = numberArrayType(nums)
	case kinds["string"]:
		elemType = "string"
	default:
		elemType = "bool"
	}
	if hasNull && elemType != "*big.Int" {
		elemType = "*" + elemType
	}
	return elemType, nil, hasNull, true
}

// nullableElem returns the element type elemType of an array that holds a
// null among objects as a pointer, so null decodes to nil and encodes back
// to null; a slice or a pointer already is nil for null.
func nullableElem(elemType string, hasNull bool) string {
	if !hasNull || strings.HasPrefix(elemType, "*") || strings.HasPrefix(elemType, "[]") {
		return elemType
	}
	return "*" + elemType
}

// mergedValues holds the values of one key in the objects of a JSON array
// (null as nil) when more than one object has the key. The field is typed
// from all of them, as the elements of an array are.
type mergedValues []interface{}

// mergeElements returns one value that stands for the elements of a JSON
// array that are not null, when they are all objects or all arrays, so
// that the element type holds every one of them: the objects merged
// (mergeObjects), or the elements of all the arrays in one array.
func mergeElements(values []interface{}) interface{} {
	var objects []*OrderedMap
	var elems []interface{}
	for _, v := range values {
		switch v := v.(type) {
		case *OrderedMap:
			objects = append(objects, v)
		case []interface{}:
			elems = append(elems, v...)
		}
	}
	if len(objects) > 0 {
		return mergeObjects(objects)
	}
	if elems == nil {
		elems = []interface{}{}
	}
	return elems
}

// mergeObjects returns one object with the keys of all the objects, in the
// order they first appear. A key that only one object has keeps its value;
// the values of a key that several objects have become one mergedValues.
// A single object is returned as it is.
func mergeObjects(objects []*OrderedMap) *OrderedMap {
	if len(objects) == 1 {
		return objects[0]
	}
	merged := &OrderedMap{
		Map:           map[string]interface{}{},
		NumberStrings: map[string]string{},
		BoolFields:    map[string]bool{},
		NullFields:    map[string]string{},
	}
	owner := map[string]*OrderedMap{}
	values := map[string][]interface{}{}
	for _, o := range objects {
		for _, k := range o.Keys {
			if _, seen := values[k]; !seen {
				merged.Keys = append(merged.Keys, k)
				owner[k] = o
			}
			values[k] = append(values[k], o.fieldValues(k, o.Map[k])...)
		}
	}
	for _, k := range merged.Keys {
		if _, ok := owner[k].Map[k].(mergedValues); ok || len(values[k]) > 1 {
			merged.Map[k] = mergedValues(values[k])
			continue
		}
		o := owner[k]
		merged.Map[k] = o.Map[k]
		if v, ok := o.NumberStrings[k]; ok {
			merged.NumberStrings[k] = v
		}
		if v, ok := o.BoolFields[k]; ok {
			merged.BoolFields[k] = v
		}
		if v, ok := o.NullFields[k]; ok {
			merged.NullFields[k] = v
		}
	}
	return merged
}

// float64Holds reports whether every number in the decoded JSON value v,
// at any depth, is one float64 holds as written: interface{} decodes them
// all into float64.
func float64Holds(v interface{}) bool {
	switch v := v.(type) {
	case json.Number:
		return float64Exact(v.String())
	case *OrderedMap:
		for _, k := range v.Keys {
			if !float64Holds(v.Map[k]) {
				return false
			}
		}
	case []interface{}:
		for _, e := range v {
			if !float64Holds(e) {
				return false
			}
		}
	case mergedValues:
		for _, e := range v {
			if !float64Holds(e) {
				return false
			}
		}
	}
	return true
}
