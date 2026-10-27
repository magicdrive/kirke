package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
)

// TestParseNumber pins the Go type chosen for each kind of JSON number.
func TestParseNumber(t *testing.T) {
	for in, want := range map[string]string{
		"0":                                "int",
		"-42":                              "int",
		"9223372036854775807":              "int",
		"9223372036854775808":              "*big.Int",
		"-9223372036854775809":             "*big.Int",
		"123456789012345678901234567890":   "*big.Int",
		"1.5":                              "float64",
		"-0.25":                            "float64",
		"0.1":                              "float64",
		"3.141592653589793238462643383279": "json.Number", // float64 rounds it
		"0.30000000000000001":              "json.Number", // float64 reads 0.3
		"1.0":                              "float64",     // the same value as 1
		"2.50":                             "float64",
		"-0.0":                             "float64",
		"12a":                              "interface{}", // not an integer
		"1.2.3":                            "interface{}", // not a float
		"-":                                "interface{}",
	} {
		if got := parseNumber(json.Number(in)); got != want {
			t.Errorf("parseNumber(%s) = %s, want %s", in, got, want)
		}
	}
}

// TestParseNumber_Exponent pins the type of JSON numbers with an exponent
// part: float64 when float64 holds the value as written, json.Number when
// it does not (overflow, underflow, digits float64 rounds away).
func TestParseNumber_Exponent(t *testing.T) {
	for in, want := range map[string]string{
		"1e3":                      "float64",
		"1E3":                      "float64",
		"1e+3":                     "float64",
		"-2e+5":                    "float64",
		"0e0":                      "float64",
		"-0e0":                     "float64",
		"1e5":                      "float64",
		"1e21":                     "float64",
		"1e0003":                   "float64",     // leading zeros in the exponent are valid JSON
		"5e-324":                   "float64",     // smallest subnormal
		"1e-400":                   "json.Number", // float64 underflows it to 0
		"-1e-99999999999999999999": "json.Number",
		"12345678901234567890e0":   "json.Number", // float64 rounds it
		"1e308":                    "float64",
		"1e309":                    "json.Number", // overflows float64
		"1e400":                    "json.Number",
		"-1e400":                   "json.Number",
		"1e999999999999":           "json.Number",
		"1e99999999999999999999":   "json.Number",
		"1.2e-4":                   "float64",
		"1.23567e100":              "float64",
		"1.23567e+100":             "float64",
		"1.7976931348623157e308":   "float64",
		"1.5e400":                  "json.Number",
		"2.5e-400":                 "json.Number",
	} {
		if got := parseNumber(json.Number(in)); got != want {
			t.Errorf("parseNumber(%s) = %s, want %s", in, got, want)
		}
	}
}

// TestParseNumber_NotJSONNumbers: strings that are not JSON numbers (the
// decoder never produces them) keep their previous result, exponent-like
// ones included.
func TestParseNumber_NotJSONNumbers(t *testing.T) {
	for _, in := range []string{
		"1e", "e5", "1e+", "1e3x", "01e3", "0x1e3", "+1e3", " 1e3", "1e3 ",
		"1,5", // a decimal comma is not a number in any locale
		"1_000", "Inf", "NaN", "", "١e٣",
	} {
		if got, want := parseNumber(json.Number(in)), legacyParseNumber(json.Number(in)); got != want {
			t.Errorf("parseNumber(%q) = %s, want %s as before", in, got, want)
		}
	}
}

// legacyParseNumber is parseNumber before exponent literals without a
// fraction were typed: the reference for every other input.
func legacyParseNumber(num json.Number) string {
	if _, err := num.Int64(); err == nil {
		return "int"
	} else if strings.Contains(num.String(), ".") {
		f, _, err := big.NewFloat(0).Parse(num.String(), 10)
		if err != nil {
			return "interface{}"
		}
		v, _ := f.Float64()
		if fmt.Sprintf("%g", v) == num.String() {
			return "float64"
		}
		return "*big.Float"
	}
	if _, ok := new(big.Int).SetString(num.String(), 10); !ok {
		return "interface{}"
	}
	return "*big.Int"
}

// FuzzParseNumber checks, for any input, that parseNumber returns a known
// type, interface{} only for what is not a JSON number; and for every JSON
// number, that encoding/json (json.Unmarshal, no UseNumber) decodes the
// literal into the chosen type and json.Marshal encodes it back to a
// number of the same value.
func FuzzParseNumber(f *testing.F) {
	for _, s := range fuzzNumberSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := parseNumber(json.Number(s))
		switch got {
		case "int", "float64", "*big.Int", "json.Number":
			if !jsonNumberGrammar.MatchString(s) {
				t.Fatalf("parseNumber(%q) = %s for what is not a JSON number", s, got)
			}
		case "interface{}":
			if jsonNumberGrammar.MatchString(s) {
				t.Fatalf("parseNumber(%q) = interface{} for a JSON number", s)
			}
			return
		default:
			t.Fatalf("parseNumber(%q) = %q, not a known type", s, got)
		}
		if strings.ContainsAny(s, ".eE") && (got == "int" || got == "*big.Int") {
			t.Fatalf("parseNumber(%q) = %s for a literal with a fraction or an exponent", s, got)
		}
		checkRoundTrip(t, got, []string{s})
	})
}

// FuzzNumberArrayType checks that the element type numberArrayType chooses
// for an array of JSON numbers, with or without a null (a pointer then),
// decodes the array with json.Unmarshal and encodes back every number with
// its value.
func FuzzNumberArrayType(f *testing.F) {
	for i, s := range fuzzNumberSeeds {
		f.Add(s, fuzzNumberSeeds[(i+7)%len(fuzzNumberSeeds)], i%2 == 0)
	}
	f.Fuzz(func(t *testing.T, a, b string, withNull bool) {
		if !jsonNumberGrammar.MatchString(a) || !jsonNumberGrammar.MatchString(b) {
			return
		}
		values := []interface{}{json.Number(a), json.Number(b)}
		if withNull {
			values = append(values, nil)
		}
		elem, _, ok := scalarArrayElemType(values, "any")
		if !ok {
			t.Fatalf("scalarArrayElemType(%v) is not a scalar type", values)
		}
		if ptr := strings.HasPrefix(elem, "*") && elem != "*big.Int"; withNull && !strings.HasPrefix(elem, "*") || !withNull && ptr {
			t.Fatalf("scalarArrayElemType(%v) = %s: a pointer exactly when null is an element", values, elem)
		}
		checkRoundTrip(t, elem, []string{a, b}, values[2:]...)
	})
}

var fuzzNumberSeeds = []string{
	"0", "-0", "1", "-42", "9223372036854775807", "9223372036854775808", "-9223372036854775809",
	"1.5", "1.0", "-0.0", "0.1", "2.50", "3.141592653589793238462643383279", "0.30000000000000001",
	"1e3", "1E3", "1.2e-4", "-2e+5", "0e0", "1e5", "1e+21", "1e400", "1.5e400", "1e-400", "2.5e-400",
	"5e-324", "1e309", "1e999999999999", "-1e-99999999999999999999", "12345678901234567890e0",
	"9007199254740993", "1.7976931348623157e308", "12a", "1,5", "+1e3", "",
}

// checkRoundTrip decodes the JSON array of literals nums (and a null when
// null is given) with json.Unmarshal into a slice of the Go type typ, which
// must succeed, and checks that json.Marshal encodes back every number
// with its value and the null as null.
func checkRoundTrip(t *testing.T, typ string, nums []string, null ...interface{}) {
	t.Helper()
	elem := typ
	ptr := false
	if strings.HasPrefix(elem, "*") && elem != "*big.Int" {
		ptr, elem = true, elem[1:]
	}
	var rt reflect.Type
	switch elem {
	case "int":
		rt = reflect.TypeOf(0)
	case "float64":
		rt = reflect.TypeOf(0.0)
	case "*big.Int":
		rt = reflect.TypeOf(new(big.Int))
	case "json.Number":
		rt = reflect.TypeOf(json.Number(""))
	default:
		t.Fatalf("%s is not a number type", typ)
	}
	if ptr {
		rt = reflect.PointerTo(rt)
	}
	doc := "[" + strings.Join(nums, ",")
	if len(null) > 0 {
		doc += ",null"
	}
	doc += "]"
	v := reflect.New(reflect.SliceOf(rt))
	if err := json.Unmarshal([]byte(doc), v.Interface()); err != nil {
		t.Fatalf("%s as []%s: encoding/json cannot decode it: %v", doc, typ, err)
	}
	out, err := json.Marshal(v.Elem().Interface())
	if err != nil {
		t.Fatalf("%s as []%s: %v", doc, typ, err)
	}
	var back []json.RawMessage
	if err := json.Unmarshal(out, &back); err != nil || len(back) != len(nums)+len(null) {
		t.Fatalf("%s as []%s encodes as %s", doc, typ, out)
	}
	for i, n := range nums {
		if !sameDecimal(n, string(back[i])) {
			t.Fatalf("%s as []%s encodes as %s: %s became %s", doc, typ, out, n, back[i])
		}
	}
	if len(null) > 0 && string(back[len(nums)]) != "null" {
		t.Fatalf("%s as []%s encodes as %s: null became %s", doc, typ, out, back[len(nums)])
	}
}

// TestApply_NumberTypes checks the number types through the decoder, in
// both output modes, for object fields and array elements: float64 when it
// holds the value as written, json.Number otherwise; never *big.Float or
// interface{}, which encoding/json cannot decode a JSON number into
// (*big.Float) or decodes only as a float64 (interface{}).
func TestApply_NumberTypes(t *testing.T) {
	in := `{"i": 1, "bi": 9223372036854775808, "f": 1.5, "bf": 1.0,` +
		` "e": 1e3, "eu": 1E3, "ef": 1.2e-4, "en": -2e+5, "ez": 0e0, "ec": 1e+21,` +
		` "o": 1e400, "u": 1e-400, "pi": 3.141592653589793238462643383279, "nested": {"e": 1e5},` +
		` "ai": [1, 2], "af": [1, 2.5], "abi": [1, 9223372036854775808], "an": [0.1, 1e400],` +
		` "ani": [1, null], "anf": [null, 1.0], "ann": [1e400, null], "anbi": [null, 9223372036854775808],` +
		` "anull": [null], "amix": [1, "a"], "anest": [[1.0, 2]], "aobj": [null, {"x": 1.0}]}`
	want := []string{
		"I int `json:\"i\"`",
		"Bi *big.Int `json:\"bi\"`",
		"F float64 `json:\"f\"`",
		"Bf float64 `json:\"bf\"`",
		"E float64 `json:\"e\"`",
		"Eu float64 `json:\"eu\"`",
		"Ef float64 `json:\"ef\"`",
		"En float64 `json:\"en\"`",
		"Ez float64 `json:\"ez\"`",
		"Ec float64 `json:\"ec\"`",
		"O json.Number `json:\"o\"`",
		"U json.Number `json:\"u\"`",
		"Pi json.Number `json:\"pi\"`",
		"Ai []int `json:\"ai\"`",
		"Af []float64 `json:\"af\"`",
		"Abi []*big.Int `json:\"abi\"`",
		"An []json.Number `json:\"an\"`",
		"Ani []*int `json:\"ani\"`",
		"Anf []*float64 `json:\"anf\"`",
		"Ann []*json.Number `json:\"ann\"`",
		"Anbi []*big.Int `json:\"anbi\"`",
		"Anull []any `json:\"anull\"`",
		"Amix []interface{} `json:\"amix\"`",
		"Anest [][]float64 `json:\"anest\"`",
		"X float64 `json:\"x\"`",
	}
	for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
		out, err := Apply(in, "Root", mode, false, "any")
		if err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		fields := map[string]bool{}
		for _, l := range strings.Split(out, "\n") {
			fields[strings.Join(strings.Fields(l), " ")] = true
		}
		for _, w := range want {
			if !fields[w] {
				t.Errorf("mode %d: no field %q in\n%s", mode, w, out)
			}
		}
		if !strings.Contains(out, "E float64 `json:\"e\"`") && !fields["E float64 `json:\"e\"`"] {
			t.Errorf("mode %d: nested exponent not float64:\n%s", mode, out)
		}
	}
}

// TestApply_DecimalComma: a decimal comma (as some locales write numbers)
// is not JSON; it is rejected, not read as a number.
func TestApply_DecimalComma(t *testing.T) {
	if out, err := Apply(`{"a": 1,5}`, "Root", commandline.OutputModeOutline, false, "any"); err == nil {
		t.Fatalf("want an error, got\n%s", out)
	}
}
