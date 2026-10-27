package core

import (
	"encoding/json"
	"strings"
)

// scalarArrayElemType returns the element type of a JSON array from all of
// its elements, as decodeJSONArray decodes them (json.Number, string, bool,
// nil, *OrderedMap, []interface{}). ok is false when the elements that are
// not null are all objects or all arrays: the caller then types the array
// from representative, its first element that is not null, and makes an
// object element a pointer when hasNull.
//
//   - no element but null: nullTypeName;
//   - elements of more than one kind (a number and a string, an object and
//     a number): interface{}, or json.RawMessage when one of them is a
//     number float64 does not hold as written (interface{} would decode it
//     into a float64: 1e400 fails, 1e-400 becomes 0);
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
		for _, n := range nums {
			if !float64Exact(n.String()) {
				return "json.RawMessage", nil, hasNull, true
			}
		}
		return "interface{}", nil, hasNull, true
	case kinds["object"], kinds["array"], kinds["other"]:
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
