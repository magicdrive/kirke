package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
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
		"3.141592653589793238462643383279": "*big.Float",
		"1.0":                              "*big.Float",  // %g prints 1, not 1.0
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
// part. Without a fraction (previously interface{}): float64 when float64
// holds the value (strconv.ParseFloat without a range error), interface{}
// otherwise; both decode with encoding/json. With a fraction: unchanged.
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
		"1e309":                  "interface{}", // overflows float64
		"1e400":                  "interface{}",
		"-1e400":                 "interface{}",
		"1e999999999999":         "interface{}",
		"1e99999999999999999999": "interface{}",
		// With a fraction: unchanged (*big.Float is the separate P1 question).
		"1.2e-4":       "*big.Float",
		"1.23567e100":  "*big.Float",
		"1.23567e+100": "float64",
		"1.5e400":      "*big.Float",
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
// type and differs from the previous behavior only by typing as float64 what
// was interface{}; and for every valid JSON number, that encoding/json
// decodes the literal into the chosen type, except *big.Float (a separate,
// pre-existing issue, only for literals with a fraction).
func FuzzParseNumber(f *testing.F) {
	for _, s := range []string{
		"0", "-0", "1", "-42", "9223372036854775807", "9223372036854775808",
		"1.5", "1.0", "-0.0", "0.1", "3.141592653589793238462643383279",
		"1e3", "1E3", "1.2e-4", "-2e+5", "0e0", "1e5", "1e+21", "1e400",
		"1e-400", "5e-324", "1e309", "1e999999999999", "12a", "1,5", "+1e3", "",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := parseNumber(json.Number(s))
		switch got {
		case "int", "float64", "*big.Int", "*big.Float", "interface{}":
		default:
			t.Fatalf("parseNumber(%q) = %q, not a known type", s, got)
		}
		if old := legacyParseNumber(json.Number(s)); got != old && (old != "interface{}" || got != "float64") {
			t.Fatalf("parseNumber(%q) = %s, was %s: only interface{} may become float64", s, got, old)
		}
		if !jsonNumberGrammar.MatchString(s) {
			return
		}
		if strings.ContainsAny(s, "eE") && (got == "int" || got == "*big.Int") {
			t.Fatalf("parseNumber(%q) = %s for a literal with an exponent", s, got)
		}
		doc := []byte(`{"N":` + s + `}`)
		var err error
		switch got {
		case "int":
			var v struct{ N int }
			err = json.Unmarshal(doc, &v)
		case "float64":
			var v struct{ N float64 }
			err = json.Unmarshal(doc, &v)
		case "*big.Int":
			var v struct{ N *big.Int }
			err = json.Unmarshal(doc, &v)
		case "interface{}":
			// Beyond float64 only a decoder with UseNumber holds it (as
			// json.Number); plain json.Unmarshal fails, as it did before.
			var v struct{ N interface{} }
			dec := json.NewDecoder(bytes.NewReader(doc))
			dec.UseNumber()
			err = dec.Decode(&v)
		case "*big.Float":
			if !strings.Contains(s, ".") {
				t.Fatalf("parseNumber(%q) = *big.Float for a literal without a fraction", s)
			}
		}
		if err != nil {
			t.Fatalf("parseNumber(%q) = %s, but encoding/json cannot decode it into one: %v", s, got, err)
		}
	})
}

// The JSON number grammar (RFC 8259, section 6).
var jsonNumberGrammar = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// TestApply_NumberTypes checks the number types through the decoder, in
// both output modes: exponent literals without a fraction are float64 (or
// interface{} beyond float64), and the other results do not change.
func TestApply_NumberTypes(t *testing.T) {
	in := `{"i": 1, "bi": 9223372036854775808, "f": 1.5, "bf": 1.0,` +
		` "e": 1e3, "eu": 1E3, "ef": 1.2e-4, "en": -2e+5, "ez": 0e0, "ec": 1e+21,` +
		` "o": 1e400, "u": 1e-400, "nested": {"e": 1e5}}`
	want := []string{
		"I int `json:\"i\"`",
		"Bi *big.Int `json:\"bi\"`",
		"F float64 `json:\"f\"`",
		"Bf *big.Float `json:\"bf\"`",
		"E float64 `json:\"e\"`",
		"Eu float64 `json:\"eu\"`",
		"Ef *big.Float `json:\"ef\"`",
		"En float64 `json:\"en\"`",
		"Ez float64 `json:\"ez\"`",
		"Ec float64 `json:\"ec\"`",
		"O interface{} `json:\"o\"`",
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
