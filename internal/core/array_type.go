package core

import (
	"encoding/json"
	"strconv"
)

// overflowArrayElemType returns the element type of a JSON array that holds
// a number beyond float64 (1e400) among its elements, as decodeJSONArray
// decodes them; ok is false for any other array, typed as before. kirke
// used to fail on such an array (its elements were decoded as float64), so
// it had no type before:
//
//   - numbers, and nulls: json.Number, *json.Number when null is among
//     them (null then decodes to nil and encodes back to null);
//   - numbers and other kinds of values: json.RawMessage, which holds any
//     JSON value as written (interface{} would decode the number into a
//     float64, which json.Unmarshal rejects).
func overflowArrayElemType(values []interface{}) (elemType string, ok bool) {
	overflow, onlyNumbers, hasNull := false, true, false
	for _, v := range values {
		switch v := v.(type) {
		case nil:
			hasNull = true
		case json.Number:
			if _, err := strconv.ParseFloat(v.String(), 64); err != nil {
				overflow = true
			}
		default:
			onlyNumbers = false
		}
	}
	switch {
	case !overflow:
		return "", false
	case !onlyNumbers:
		return "json.RawMessage", true
	case hasNull:
		return "*json.Number", true
	default:
		return "json.Number", true
	}
}

// firstNonNull returns the first element of values that is not null, nil
// when there is none.
func firstNonNull(values []interface{}) interface{} {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
