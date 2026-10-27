package core

import "encoding/json"

// scalarArrayElemType returns the element type of a JSON array from all of
// its elements, as decodeJSONArray decodes them (json.Number, string, bool,
// nil, *OrderedMap, []interface{}). ok is false when the elements that are
// not null are objects or arrays, all of one kind: the caller then types
// the array from representative, its first element that is not null.
//
//   - no element but null: nullTypeName;
//   - elements of more than one kind (a number and a string, an object
//     and a number): interface{};
//   - numbers: numberArrayType of all of them;
//   - strings: string; booleans: bool;
//   - a null among numbers, strings or booleans makes the element a
//     pointer (*int, *float64, *json.Number, *string, *bool), so null
//     decodes to nil and encodes back to null; *big.Int already is one.
func scalarArrayElemType(values []interface{}, nullTypeName string) (elemType string, representative interface{}, ok bool) {
	var nums []json.Number
	kinds := map[string]bool{}
	hasNull := false
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
		return nullTypeName, nil, true
	case len(kinds) > 1:
		return "interface{}", nil, true
	case kinds["object"], kinds["array"], kinds["other"]:
		return "", representative, false
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
	return elemType, nil, true
}
