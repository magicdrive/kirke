package main

// End-to-end round trip through the generated code of the JSON values whose
// generated type could not decode them (*big.Float, interface{} beyond
// float64) or that kirke failed or panicked on: a JSON document is decoded
// into the struct kirke generates for it, encoded again with json.Marshal,
// and every value of the result must be the one it came from (numbers by
// value), in both output modes and with both json.Unmarshal and a
// json.Decoder using UseNumber.

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/core"
)

func TestGeneratedStruct_NumberRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Go programs")
	}
	for _, tc := range []struct {
		name string
		json string
		// Field declarations (gofmt spacing collapsed) the generated code
		// must contain, in both output modes unless inline is set.
		fields []string
		inline bool // inline mode only
	}{
		{
			name: "fractions that were *big.Float",
			json: `{"one": 1.0, "trailing": 2.50, "negfraczero": -0.0, "expfrac": 1.2e-4, "big": 1.23567e100,` +
				` "pi": 3.141592653589793238462643383279, "rounded": 0.30000000000000001, "fracover": 1.5e400,` +
				` "fracunder": -2.5e-400, "huge": 1.5e99999999999999999999, "nested": {"x": 1.0, "y": 1.5e400}}`,
			fields: []string{"One float64", "Trailing float64", "Negfraczero float64", "Expfrac float64",
				"Big float64", "Pi json.Number", "Rounded json.Number", "Fracover json.Number",
				"Fracunder json.Number", "Huge json.Number", "X float64", "Y json.Number"},
		},
		{
			name: "beyond float64, was interface{}",
			json: `{"overflow": 1e400, "negoverflow": -1e309, "hugeexp": 1e99999999999999999999,` +
				` "objs": [{"v": 1e400}]}`,
			fields: []string{"Overflow json.Number", "Negoverflow json.Number", "Hugeexp json.Number", "V json.Number"},
		},
		{
			name: "arrays beyond float64, kirke failed",
			json: `{"far": [1e400, 1], "farnull": [null, -1e400, 2.5], "nested": [[1e400]],` +
				` "deep": {"x": [1.5e400]}, "mixed": [1e400, "a"]}`,
			fields: []string{"Far []json.Number", "Farnull []*json.Number", "Nested [][]json.Number",
				"X []json.Number", "Mixed []json.RawMessage"},
		},
		{
			name: "arrays starting with null, inline panicked",
			json: `{"num": [null, 1, 2.5], "obj": [null, {"x": 1.0}], "str": [null, "a"], "arr": [null, [1]],` +
				` "nulls": [null, null]}`,
			fields: []string{"Num []*float64", "Obj []*struct", "Str []*string", "Arr [][]int",
				"Nulls []any"},
			inline: true,
		},
		// Contract changes: values float64 changes become json.Number, arrays
		// are typed from all of their elements.
		{
			name: "integers",
			json: `{"int": 42, "zero": 0, "neg": -7, "negzero": -0, "maxint": 9223372036854775807,` +
				` "minint": -9223372036854775808, "bigint": 123456789012345678901234567890,` +
				` "negbigint": -9223372036854775809}`,
			fields: []string{"Int int", "Zero int", "Maxint int", "Bigint *big.Int", "Negbigint *big.Int"},
		},
		{
			name: "fractions",
			json: `{"frac": 1.5, "tenth": 0.1, "negfrac": -0.25, "fraczero": 0.0,` +
				` "small": 0.000000000000000000000000000001, "underflow": 1.5e-99999999999999999999}`,
			fields: []string{"Frac float64", "Tenth float64", "Fraczero float64", "Small float64",
				"Underflow json.Number"},
		},
		{
			name: "exponents float64 changes",
			json: `{"exp": 1e3, "negexp": -2.5E+10, "tiny": 5e-324, "max": 1.7976931348623157e308,` +
				` "underflow": 1e-400, "negunderflow": -1e-99999999999999999999, "inexact": 12345678901234567890e0}`,
			fields: []string{"Exp float64", "Negexp float64", "Tiny float64", "Max float64",
				"Underflow json.Number", "Negunderflow json.Number", "Inexact json.Number"},
		},
		{
			name: "arrays from all elements",
			json: `{"ints": [1, 2, 3], "nums": [1, 2.5, -3e2], "bigints": [1, 9223372036854775808],` +
				` "lossy": [0.1, 3.141592653589793238462643383279], "under": [1e-400, 1],` +
				` "inexactint": [9007199254740993, 0.5], "matrix": [[1, 2], [3]], "fmatrix": [[1.0], [2.5]]}`,
			fields: []string{"Ints []int", "Nums []float64", "Bigints []*big.Int", "Lossy []json.Number",
				"Under []json.Number", "Inexactint []json.Number", "Matrix [][]int", "Fmatrix [][]float64"},
		},
		{
			name: "nullable",
			json: `{"ints": [1, null, 3], "floats": [null, 1.5], "far": [null, 1e400],` +
				` "bigints": [9223372036854775808, null], "strs": ["a", null], "bools": [true, null],` +
				` "nulls": [null], "null": null, "objs": [{"x": 1}, null], "arrs": [[1], null]}`,
			fields: []string{"Ints []*int", "Floats []*float64", "Far []*json.Number",
				"Bigints []*big.Int", "Strs []*string", "Bools []*bool", "Nulls []any", "Null any",
				"Arrs [][]int"},
		},
		{
			name: "nested and mixed",
			json: `{"nested": {"x": 1.0, "deep": {"y": 1e400, "z": [2.5, null]}},` +
				` "objs": [{"v": 1.0, "w": 1e400}, {"v": 2.0, "w": 3}], "mixed": [1, "a", true, null],` +
				` "mixedlossy": [1e-400, "a"], "s": "str", "b": true}`,
			fields: []string{"X float64", "Y json.Number", "Z []*float64", "V float64", "W json.Number",
				"Mixed []interface{}", "Mixedlossy []json.RawMessage"},
		},
	} {
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			if tc.inline && mode != commandline.OutputModeInline {
				continue
			}
			t.Run(tc.name+map[int]string{commandline.OutputModeOutline: "/outline", commandline.OutputModeInline: "/inline"}[mode], func(t *testing.T) {
				src, err := core.Apply(tc.json, "AutoGenerated", mode, false, "any")
				if err != nil {
					t.Fatal(err)
				}
				fields := map[string]bool{}
				for _, l := range strings.Split(src, "\n") {
					f := strings.Fields(l)
					if len(f) >= 2 {
						fields[f[0]+" "+f[1]] = true
					}
				}
				for _, w := range tc.fields {
					if !fields[w] {
						t.Errorf("no field %q in\n%s", w, src)
					}
				}
				bin := buildGenerated(t, src)
				for _, decode := range []string{"unmarshal", "usenumber"} {
					out := runBinary(t, bin, src, tc.json, decode)
					if err := sameJSON(tc.json, out); err != nil {
						t.Errorf("%s: %s re-encoded as %s: %v\n%s", decode, tc.json, out, err, src)
					}
				}
			})
		}
	}
}

// TestGeneratedStruct_MergedArrayElements: an array's objects, which can
// have different keys, decode into the struct generated from all of them
// without losing a key (DisallowUnknownFields) or a value: every value of
// the document comes back from json.Marshal (numbers by value). A key an
// object did not have comes back with its zero value, which is not
// checked. Arrays of arrays decode every element, and so do mixed arrays
// with numbers inside their objects and arrays; outline structs that
// share a name compile.
func TestGeneratedStruct_MergedArrayElements(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Go programs")
	}
	for _, tc := range []struct{ name, json string }{
		{"numbers of one key", `{"objs": [{"value": 1}, {"value": 2.5}, {"value": 9007199254740993}],` +
			` "big": [{"n": 1}, {"n": 18446744073709551616}], "lossy": [{"n": 1}, {"n": 1e-400}]}`},
		{"different keys", `{"objs": [{"a": 1}, {"b": "hello"}, {"c": true, "a": -2}]}`},
		{"nested objects", `{"objs": [{"nested": {"a": 1}}, {"nested": {"b": 2.5, "c": [1, null]}}]}`},
		{"arrays of arrays", `{"arrs": [[1, 2], [3, 4.5]], "deep": [[[1]], [[2.5, null]], []],` +
			` "objs": [[{"x": 1}], [{"y": [{"z": 1}, {"z": 1e400}]}]]}`},
		{"null and mixed in a key", `{"objs": [{"a": 1, "o": {"p": 1}, "m": 1, "l": 1e-400}, null,` +
			` {"a": null, "o": null, "m": "s", "l": "t"}, {"m": null, "n": null}]}`},
		// A number float64 changes inside an object or array of a mixed
		// array: interface{} decoded it into a float64, without an error.
		{"mixed with nested numbers", `{"x": ["s", {"a": 9007199254740993}], "y": [1, [0.30000000000000001]],` +
			` "z": ["s", [1e-400]], "w": [true, {"a": 18446744073709551616}], "v": [{"a": [1e400]}, 1],` +
			` "u": ["s", {"a": 1.5}]}`},
		// Outline structs with the same name: they were all declared, and
		// the output did not compile.
		// Keys merging brings in: names that are not identifiers, that
		// clash, and empty objects (inline struct{}).
		{"field names and empty objects", `{"ops": [{"a": 1}, {"$set": {"x": 1}, "a_b": 2, "aB": "s"}],` +
			` "e": [{"x": null}, {"x": {}}], "dup": {"k": 1, "k": "last"}, "dash": {"-": 1},` +
			` "sym": {"a b": 1, "@type": "t"}, "名前": {"値": 1, "_": 2}}`},
		{"struct names", `{"x": {"a": [null, {"b": 1}]}, "y": {"a": [{"c": 1}]}, "meta": {"meta": {"m": 1}},` +
			` "auto_generated": {"r": 1}, "p": {"c": {"d": 1}}, "q": {"c": {"d": 2}}}`},
	} {
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			t.Run(tc.name+map[int]string{commandline.OutputModeOutline: "/outline", commandline.OutputModeInline: "/inline"}[mode], func(t *testing.T) {
				src, err := core.Apply(tc.json, "AutoGenerated", mode, false, "any")
				if err != nil {
					t.Fatal(err)
				}
				bin := buildGenerated(t, src)
				for _, decode := range []string{"unmarshal", "strict"} {
					out := runBinary(t, bin, src, tc.json, decode)
					if err := containsJSON(tc.json, out); err != nil {
						t.Errorf("%s: %s re-encoded as %s: %v\n%s", decode, tc.json, out, err, src)
					}
				}
			})
		}
	}
}

// containsJSON reports a value of the JSON document a that b does not
// have at the same place: as sameJSON, except that an object of b may have
// keys a does not have.
func containsJSON(a, b string) error {
	decode := func(s string) (interface{}, error) {
		dec := json.NewDecoder(strings.NewReader(s))
		dec.UseNumber()
		var v interface{}
		err := dec.Decode(&v)
		return v, err
	}
	av, err := decode(a)
	if err != nil {
		return err
	}
	bv, err := decode(b)
	if err != nil {
		return err
	}
	return compareValue("$", av, bv, true)
}

// sameJSON reports how the JSON documents a and b differ: numbers are
// compared by value (1.0 and 1, -0 and 0 are the same), the rest exactly.
func sameJSON(a, b string) error {
	decode := func(s string) (interface{}, error) {
		dec := json.NewDecoder(strings.NewReader(s))
		dec.UseNumber()
		var v interface{}
		err := dec.Decode(&v)
		return v, err
	}
	av, err := decode(a)
	if err != nil {
		return err
	}
	bv, err := decode(b)
	if err != nil {
		return err
	}
	return compareValue("$", av, bv, false)
}

// compareValue reports where b differs from a; with subset, an object of b
// may have more keys than the object of a.
func compareValue(path string, a, b interface{}, subset bool) error {
	switch a := a.(type) {
	case map[string]interface{}:
		b, ok := b.(map[string]interface{})
		if !ok || len(a) > len(b) || !subset && len(a) != len(b) {
			return fmt.Errorf("%s: %v, then %v", path, a, b)
		}
		for k := range a {
			if _, ok := b[k]; !ok {
				return fmt.Errorf("%s: no key %q in %v", path, k, b)
			}
			if err := compareValue(path+"."+k, a[k], b[k], subset); err != nil {
				return err
			}
		}
		return nil
	case []interface{}:
		b, ok := b.([]interface{})
		if !ok || len(a) != len(b) {
			return fmt.Errorf("%s: %v, then %v", path, a, b)
		}
		for i := range a {
			if err := compareValue(fmt.Sprintf("%s[%d]", path, i), a[i], b[i], subset); err != nil {
				return err
			}
		}
		return nil
	case json.Number:
		b, ok := b.(json.Number)
		if !ok || !sameNumber(a.String(), b.String()) {
			return fmt.Errorf("%s: number %s, then %v", path, a, b)
		}
		return nil
	default:
		if fmt.Sprint(a) != fmt.Sprint(b) {
			return fmt.Errorf("%s: %v, then %v", path, a, b)
		}
		return nil
	}
}

// sameNumber compares two JSON number literals by value. A literal with an
// exponent of more than four digits (1e99999999999999999999, beyond what
// big.Rat should expand) must come back as written.
func sameNumber(a, b string) bool {
	if a == b {
		return true
	}
	if longExponent.MatchString(a) || longExponent.MatchString(b) {
		return false
	}
	ar, ok1 := new(big.Rat).SetString(a)
	br, ok2 := new(big.Rat).SetString(b)
	return ok1 && ok2 && ar.Cmp(br) == 0
}

var longExponent = regexp.MustCompile(`[eE][+-]?[0-9]{5,}$`)
