package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type OrderedMap struct {
	Keys          []string
	Map           map[string]interface{}
	NumberStrings map[string]string
	BoolFields    map[string]bool
	NullFields    map[string]string
}

func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	o.Map = make(map[string]interface{})
	o.Keys = nil
	o.NumberStrings = make(map[string]string)
	o.BoolFields = make(map[string]bool)
	o.NullFields = make(map[string]string)

	nullValue := "nil"

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{', but got %v", tok)
	}

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return fmt.Errorf("expected string for key, but got %v", keyTok)
		}
		// A key that appears again is one field, typed from all of its
		// values (encoding/json decodes each of them into it); it was a
		// second field of the same name, which did not compile.
		var earlier mergedValues
		if prev, seen := o.Map[key]; seen {
			earlier = o.fieldValues(key, prev)
			delete(o.NumberStrings, key)
			delete(o.BoolFields, key)
			delete(o.NullFields, key)
		} else {
			o.Keys = append(o.Keys, key)
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}

		rawStr := strings.TrimSpace(string(raw))
		if len(rawStr) > 0 && (rawStr[0] == '"' || rawStr[0] == '{' || rawStr[0] == '[') {
			var value interface{}
			if isJSONArray(raw) {
				var arrayValues []interface{}
				if err := decodeJSONArray(raw, &arrayValues); err != nil {
					return err
				}
				o.Map[key] = arrayValues
			} else if isJSONObject(raw) {
				nestedMap := &OrderedMap{}
				if err := nestedMap.UnmarshalJSON(raw); err != nil {
					return err
				}
				o.Map[key] = nestedMap
			} else {
				if err := json.Unmarshal(raw, &value); err != nil {
					return err
				}
				o.Map[key] = value
			}
		} else if rawStr == "true" || rawStr == "false" {
			boolValue := rawStr == "true"
			o.BoolFields[key] = boolValue
			o.Map[key] = boolValue
		} else if rawStr == "null" {
			o.NullFields[key] = nullValue
			o.Map[key] = nullValue
		} else {
			o.NumberStrings[key] = rawStr
			o.Map[key] = json.Number(rawStr)
		}
		if earlier != nil {
			value := o.fieldValues(key, o.Map[key])
			delete(o.NumberStrings, key)
			delete(o.BoolFields, key)
			delete(o.NullFields, key)
			o.Map[key] = append(earlier, value...)
		}
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

// fieldValues returns the values the field key holds: those of a key that
// appears more than once, or its value (nil for null).
func (o *OrderedMap) fieldValues(key string, value interface{}) mergedValues {
	if v, ok := value.(mergedValues); ok {
		return v
	}
	if _, isNull := o.NullFields[key]; isNull {
		return mergedValues{nil}
	}
	return mergedValues{value}
}

// decodeJSONArray decodes the JSON array b into result: an object as an
// *OrderedMap, an array as a []interface{}, a number as a json.Number (its
// literal: a number beyond float64 such as 1e400 does not fail, as it did
// when numbers were decoded as float64), and a string, a boolean or null as
// encoding/json decodes them.
func decodeJSONArray(b json.RawMessage, result *[]interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if _, err := dec.Token(); err != nil {
		return err
	}

	for dec.More() {
		var rawItem json.RawMessage
		if err := dec.Decode(&rawItem); err != nil {
			return err
		}

		var item interface{}
		if isJSONObject(rawItem) {
			nestedMap := &OrderedMap{}
			if err := nestedMap.UnmarshalJSON(rawItem); err != nil {
				return err
			}
			item = nestedMap
		} else if isJSONArray(rawItem) {
			var nestedArray []interface{}
			if err := decodeJSONArray(rawItem, &nestedArray); err != nil {
				return err
			}
			item = nestedArray
		} else {
			itemDec := json.NewDecoder(bytes.NewReader(rawItem))
			itemDec.UseNumber()
			if err := itemDec.Decode(&item); err != nil {
				return err
			}
		}
		*result = append(*result, item)
	}

	if _, err := dec.Token(); err != nil {
		return err
	}

	return nil
}

func isJSONObject(b json.RawMessage) bool {
	b = bytes.TrimSpace(b)
	return len(b) > 0 && b[0] == '{' && b[len(b)-1] == '}'
}

func isJSONArray(b json.RawMessage) bool {
	b = bytes.TrimSpace(b)
	return len(b) > 0 && b[0] == '[' && b[len(b)-1] == ']'
}
