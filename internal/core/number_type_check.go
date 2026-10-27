package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// jsonExponentInteger matches a JSON number with an exponent part and no
// fraction (RFC 8259, section 6): 1e3, 1E3, -2e+5, 0e0.
var jsonExponentInteger = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)[eE][+-]?[0-9]+$`)

// parseNumber returns the Go type for a JSON number literal:
//
//   - an integer (no fraction, no exponent) that fits in int64: int;
//   - any other integer: *big.Int;
//   - an exponent and no fraction (1e3, 1E3, -2e+5, 0e0): float64 when
//     strconv.ParseFloat parses it without a range error (underflow to
//     zero or a subnormal included, as encoding/json decodes it), else
//     interface{} (overflow such as 1e400), which decodes any number;
//   - a fraction, with or without an exponent: float64 when %g of its
//     float64 value prints the literal exactly (1.5, 0.1, 1.23567e+100),
//     *big.Float otherwise (1.0, 1.2e-4);
//   - interface{} when it cannot be parsed.
//
// A literal with an exponent is never int or *big.Int, even when its value
// is an integer: encoding/json and (*big.Int).UnmarshalJSON reject it for
// those types. Parsing does not depend on the locale.
func parseNumber(num json.Number) string {
	if jsonExponentInteger.MatchString(num.String()) {
		if _, err := strconv.ParseFloat(num.String(), 64); err != nil {
			// Out of the float64 range.
			return "interface{}"
		}
		return "float64"
	}
	if _, err := num.Int64(); err == nil {
		return "int"
	} else if strings.Contains(num.String(), ".") {
		bigFloatValue, _, parseErr := big.NewFloat(0).Parse(num.String(), 10)
		if parseErr != nil {
			//Error converting to big.Float
			return "interface{}"
		}

		floatValue, _ := bigFloatValue.Float64()
		floatStr := fmt.Sprintf("%g", floatValue)

		if floatStr == num.String() {
			return "float64"
		} else {
			return "*big.Float"
		}
	} else {
		// Only whether it parses matters: the type is *big.Int either way.
		if _, ok := new(big.Int).SetString(num.String(), 10); !ok {
			//Error converting to big.Int:
			return "interface{}"
		}
		return "*big.Int"
	}
}
