package core

import (
	"encoding/json"
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
// part. They follow the rule of the numbers with a fraction: float64 when
// %g of the float64 value prints the literal exactly, *big.Float otherwise
// (overflow and underflow included), interface{} only when big.Float
// cannot parse the literal. Never int or *big.Int, which encoding/json
// cannot decode such a literal into.
func TestParseNumber_Exponent(t *testing.T) {
	for in, want := range map[string]string{
		"1e3":                     "*big.Float", // %g prints 1000
		"1E3":                     "*big.Float",
		"1e+3":                    "*big.Float",
		"-2e+5":                   "*big.Float",
		"0e0":                     "*big.Float", // %g prints 0, as for 0.0
		"-0e0":                    "*big.Float",
		"1e5":                     "*big.Float",
		"1e21":                    "*big.Float", // %g prints 1e+21
		"1e0003":                  "*big.Float", // leading zeros in the exponent are valid JSON
		"1e+21":                   "float64",
		"-1e+21":                  "float64",
		"1e-07":                   "float64",
		"2e+06":                   "float64",
		"5e-324":                  "float64",    // smallest subnormal
		"1e400":                   "*big.Float", // overflows float64
		"-1e400":                  "*big.Float",
		"1e-400":                  "*big.Float", // underflows float64
		"12345678901234567890e+0": "*big.Float",
		"1e999999999999":          "interface{}", // beyond big.Float's exponent range
		"1e99999999999999999999":  "interface{}",
		// With a fraction too: unchanged.
		"1.2e-4":                  "*big.Float", // %g prints 0.00012
		"1.23567e100":             "*big.Float",
		"1.23567e+100":            "float64",
		"1.7976931348623157e+308": "float64", // largest float64
		"1.8e+308":                "*big.Float",
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
	for in, want := range map[string]string{
		"1e":    "interface{}",
		"e5":    "interface{}",
		"1e+":   "interface{}",
		"1e3x":  "interface{}",
		"01e3":  "interface{}",
		"0x1e3": "interface{}",
		"+1e3":  "interface{}",
		" 1e3":  "interface{}",
		"1e3 ":  "interface{}",
		"1,5":   "interface{}", // a decimal comma is not a number in any locale
		"1_000": "interface{}",
		"Inf":   "interface{}",
		"NaN":   "interface{}",
		"":      "interface{}",
		"١e٣":   "interface{}", // non-ASCII digits
	} {
		if got := parseNumber(json.Number(in)); got != want {
			t.Errorf("parseNumber(%q) = %s, want %s", in, got, want)
		}
	}
}

// FuzzParseNumber checks, for every valid JSON number, that the chosen type
// can hold it: encoding/json decodes the literal into an int64, float64 or
// *big.Int field without error when that is the type, a literal with an
// exponent is never int or *big.Int, and interface{} is chosen only when
// big.Float cannot parse the literal. For any input it must not panic and
// must return one of the known types.
func FuzzParseNumber(f *testing.F) {
	for _, s := range []string{
		"0", "-0", "1", "-42", "9223372036854775807", "9223372036854775808",
		"1.5", "1.0", "-0.0", "0.1", "3.141592653589793238462643383279",
		"1e3", "1E3", "1.2e-4", "-2e+5", "0e0", "1e5", "1e+21", "1e400",
		"1e-400", "5e-324", "1e999999999999", "12a", "1,5", "+1e3", "",
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
		if !jsonNumberGrammar.MatchString(s) {
			return
		}
		if strings.ContainsAny(s, "eE") && (got == "int" || got == "*big.Int") {
			t.Fatalf("parseNumber(%q) = %s for a literal with an exponent", s, got)
		}
		var err error
		switch got {
		case "int":
			var v struct{ N int64 }
			err = json.Unmarshal([]byte(`{"N":`+s+`}`), &v)
		case "float64":
			var v struct{ N float64 }
			err = json.Unmarshal([]byte(`{"N":`+s+`}`), &v)
		case "*big.Int":
			var v struct{ N *big.Int }
			err = json.Unmarshal([]byte(`{"N":`+s+`}`), &v)
		case "interface{}":
			if _, _, perr := big.NewFloat(0).Parse(s, 10); perr == nil {
				t.Fatalf("parseNumber(%q) = interface{}, but big.Float parses it", s)
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
// both output modes: exponent literals are typed like the other non-integer
// literals instead of falling back to interface{}, and the existing int,
// *big.Int, float64 and *big.Float results do not change.
func TestApply_NumberTypes(t *testing.T) {
	in := `{"i": 1, "bi": 9223372036854775808, "f": 1.5, "bf": 1.0,` +
		` "e": 1e3, "eu": 1E3, "ef": 1.2e-4, "en": -2e+5, "ez": 0e0, "ec": 1e+21,` +
		` "o": 1e400, "u": 1e-400, "nested": {"e": 1e5}}`
	want := []string{
		"I int `json:\"i\"`",
		"Bi *big.Int `json:\"bi\"`",
		"F float64 `json:\"f\"`",
		"Bf *big.Float `json:\"bf\"`",
		"E *big.Float `json:\"e\"`",
		"Eu *big.Float `json:\"eu\"`",
		"Ef *big.Float `json:\"ef\"`",
		"En *big.Float `json:\"en\"`",
		"Ez *big.Float `json:\"ez\"`",
		"Ec float64 `json:\"ec\"`",
		"O *big.Float `json:\"o\"`",
		"U *big.Float `json:\"u\"`",
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
		if strings.Contains(out, "interface{}") {
			t.Errorf("mode %d: a number typed interface{}:\n%s", mode, out)
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
