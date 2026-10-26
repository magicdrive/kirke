package core

import (
	"encoding/json"
	"testing"
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
