package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
)

// TestParseNumber pins the Go type chosen for each kind of JSON number.
func TestParseNumber(t *testing.T) {
	for in, want := range map[string]string{
		"0":                              "int",
		"-42":                            "int",
		"9223372036854775807":            "int",
		"9223372036854775808":            "*big.Int",
		"-9223372036854775809":           "*big.Int",
		"123456789012345678901234567890": "*big.Int",
		"1.5":                            "float64",
		"-0.25":                          "float64",
		"0.1":                            "float64",
		// Was *big.Float, which no JSON number decodes into.
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
// part. Without a fraction: float64 when strconv.ParseFloat parses it
// without a range error (an underflow included, unchanged), json.Number
// beyond float64 (was interface{}, which json.Unmarshal rejects). With a
// fraction: float64 or json.Number instead of *big.Float.
func TestParseNumber_Exponent(t *testing.T) {
	for in, want := range map[string]string{
		"1e3":                    "float64",
		"1E3":                    "float64",
		"1e+3":                   "float64",
		"-2e+5":                  "float64",
		"0e0":                    "float64",
		"-0e0":                   "float64",
		"1e5":                    "float64",
		"1e21":                   "float64",
		"1e0003":                 "float64", // leading zeros in the exponent are valid JSON
		"5e-324":                 "float64", // smallest subnormal
		"1e-400":                 "float64", // underflows to 0, as encoding/json decodes it
		"12345678901234567890e0": "float64", // rounded, as encoding/json decodes it
		"1e308":                  "float64",
		"1e309":                  "json.Number", // overflows float64
		"1e400":                  "json.Number",
		"-1e400":                 "json.Number",
		"1e999999999999":         "json.Number",
		"1e99999999999999999999": "json.Number",
		// With a fraction: float64 when float64 holds the value as written,
		// json.Number otherwise (was *big.Float).
		"1.2e-4":                    "float64",
		"1.23567e100":               "float64",
		"1.23567e+100":              "float64", // unchanged
		"1.5e400":                   "json.Number",
		"2.5e-400":                  "json.Number",
		"1.5e99999999999999999999":  "json.Number", // was interface{}: beyond float64
		"1.5e-99999999999999999999": "interface{}", // unchanged: decodes as 0
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
// type and differs from the previous behavior (previousParseNumber) only
// where that type never decoded the literal: *big.Float became float64 when
// float64 holds the value as written, json.Number otherwise, and
// interface{} beyond float64 became json.Number. For every JSON number,
// json.Unmarshal (no UseNumber) decodes the literal into the type, and a
// json.Number or a float64 that was *big.Float encodes back with its value.
func FuzzParseNumber(f *testing.F) {
	for _, s := range fuzzNumberSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := parseNumber(json.Number(s))
		switch got {
		case "int", "float64", "*big.Int", "json.Number", "interface{}":
		default:
			t.Fatalf("parseNumber(%q) = %q, not a known type", s, got)
		}
		valid := jsonNumberGrammar.MatchString(s)
		prev := previousParseNumber(json.Number(s))
		switch {
		case got == prev:
		case prev == "*big.Float" && !valid && got == "interface{}":
			// Not a JSON number: the decoder never produces one.
		case prev == "*big.Float" && valid && (got == "float64") == float64Exact(s) && (got == "float64" || got == "json.Number"):
		case prev == "interface{}" && valid && got == "json.Number":
			if _, err := strconv.ParseFloat(s, 64); err == nil {
				t.Fatalf("parseNumber(%q) = json.Number, was interface{}, which decodes it", s)
			}
		default:
			t.Fatalf("parseNumber(%q) = %s, was %s", s, got, prev)
		}
		if !valid {
			return
		}
		if strings.ContainsAny(s, "eE") && (got == "int" || got == "*big.Int") {
			t.Fatalf("parseNumber(%q) = %s for a literal with an exponent", s, got)
		}
		out, err := decodeAndEncode(got, `{"N":`+s+`}`)
		if err != nil {
			t.Fatalf("parseNumber(%q) = %s, but encoding/json cannot decode it into one: %v", s, got, err)
		}
		if got == "json.Number" || prev == "*big.Float" {
			var back struct{ N json.RawMessage }
			if err := json.Unmarshal(out, &back); err != nil || !sameDecimal(s, string(back.N)) {
				t.Fatalf("parseNumber(%q) = %s encodes back as %s", s, got, out)
			}
		}
	})
}

// FuzzApply_Numbers checks that a JSON number decodes with json.Unmarshal
// (no UseNumber) into the type kirke generates for it, in both output
// modes, as an object field, alone in an array, after another number and
// after or before a null.
func FuzzApply_Numbers(f *testing.F) {
	for i, s := range fuzzNumberSeeds {
		f.Add(s, fuzzNumberSeeds[(i+7)%len(fuzzNumberSeeds)])
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		if !jsonNumberGrammar.MatchString(a) || !jsonNumberGrammar.MatchString(b) {
			return
		}
		doc := `{"n": ` + a + `, "l": [` + a + `], "p": [` + b + `, ` + a + `], "q": [null, ` + a + `], "r": [` + a + `, null]}`
		for _, mode := range []int{commandline.OutputModeOutline, commandline.OutputModeInline} {
			src, err := Apply(doc, "Root", mode, false, "any")
			if err != nil {
				t.Fatalf("mode %d: %s: %v", mode, doc, err)
			}
			for _, l := range strings.Split(src, "\n") {
				fields := strings.Fields(l)
				if len(fields) != 3 || fields[0] == "type" {
					continue
				}
				typ, tag := fields[1], fields[2]
				key := strings.TrimSuffix(strings.TrimPrefix(tag, "`json:\""), "\"`")
				var raw map[string]json.RawMessage
				if err := json.Unmarshal([]byte(doc), &raw); err != nil {
					t.Fatal(err)
				}
				if _, err := decodeAndEncode(typ, `{"N":`+string(raw[key])+`}`); err != nil {
					t.Fatalf("mode %d: %s: field %s %s does not decode it: %v\n%s", mode, doc, key, typ, err, src)
				}
			}
		}
	})
}

var fuzzNumberSeeds = []string{
	"0", "-0", "1", "-42", "9223372036854775807", "9223372036854775808", "-9223372036854775809",
	"1.5", "1.0", "-0.0", "0.1", "2.50", "3.141592653589793238462643383279", "0.30000000000000001",
	"1e3", "1E3", "1.2e-4", "-2e+5", "0e0", "1e5", "1e+21", "1e400", "1.5e400", "1e-400", "2.5e-400",
	"5e-324", "1e309", "1e999999999999", "1.5e99999999999999999999", "1.5e-99999999999999999999",
	"12345678901234567890e0", "9007199254740993", "1.7976931348623157e308", "12a", "1,5", "+1e3", "",
}

// decodeAndEncode decodes doc, an object with one member N, with
// json.Unmarshal into a struct whose N has the Go type typ (as kirke
// writes it), and returns it encoded again with json.Marshal.
func decodeAndEncode(typ, doc string) ([]byte, error) {
	rt, err := goType(typ)
	if err != nil {
		return nil, err
	}
	v := reflect.New(reflect.StructOf([]reflect.StructField{{Name: "N", Type: rt}}))
	if err := json.Unmarshal([]byte(doc), v.Interface()); err != nil {
		return nil, err
	}
	return json.Marshal(v.Interface())
}

// goType returns the reflect.Type of the Go type typ kirke writes for JSON
// numbers, nulls and arrays of them.
func goType(typ string) (reflect.Type, error) {
	switch typ {
	case "int":
		return reflect.TypeOf(0), nil
	case "float64":
		return reflect.TypeOf(0.0), nil
	case "*big.Int":
		return reflect.TypeOf(new(big.Int)), nil
	case "*big.Float":
		return reflect.TypeOf(new(big.Float)), nil
	case "json.Number":
		return reflect.TypeOf(json.Number("")), nil
	case "json.RawMessage":
		return reflect.TypeOf(json.RawMessage(nil)), nil
	case "interface{}", "any":
		return reflect.TypeOf((*interface{})(nil)).Elem(), nil
	}
	if strings.HasPrefix(typ, "[]") {
		elem, err := goType(typ[2:])
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(elem), nil
	}
	if strings.HasPrefix(typ, "*") {
		elem, err := goType(typ[1:])
		if err != nil {
			return nil, err
		}
		return reflect.PointerTo(elem), nil
	}
	return nil, fmt.Errorf("unknown type %s", typ)
}

// previousParseNumber is parseNumber before *big.Float and interface{}
// beyond float64 were replaced (kirke d819f82): the reference for every
// type that did decode.
func previousParseNumber(num json.Number) string {
	if jsonExponentInteger.MatchString(num.String()) {
		if _, err := strconv.ParseFloat(num.String(), 64); err != nil {
			return "interface{}"
		}
		return "float64"
	}
	return legacyParseNumber(num)
}

// TestApply_NumberTypes checks the number types through the decoder, in
// both output modes: no *big.Float, json.Number beyond float64.
func TestApply_NumberTypes(t *testing.T) {
	in := `{"i": 1, "bi": 9223372036854775808, "f": 1.5, "bf": 1.0,` +
		` "e": 1e3, "eu": 1E3, "ef": 1.2e-4, "en": -2e+5, "ez": 0e0, "ec": 1e+21,` +
		` "o": 1e400, "u": 1e-400, "nested": {"e": 1e5}}`
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
		"U float64 `json:\"u\"`",
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
