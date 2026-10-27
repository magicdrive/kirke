package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// jsonExponentNumber matches a JSON number with an exponent part
// (RFC 8259, section 6): 1e3, 1E3, -2e+5, 1.2e-4, 0e0.
var jsonExponentNumber = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?[eE][+-]?[0-9]+$`)

// parseNumber returns the Go type for a JSON number literal:
//
//   - an integer (no fraction, no exponent) that fits in int64: int;
//   - any other integer: *big.Int;
//   - a literal with a fraction or an exponent: float64 when %g of its
//     float64 value prints the literal exactly (1.5, 0.1, 1e+21, 1e-07),
//     *big.Float otherwise (1.0, 1e3, 1.2e-4, overflow such as 1e400,
//     underflow such as 1e-400, more digits than float64 holds);
//   - interface{} when it cannot be parsed (not a number, or an exponent
//     beyond the range of big.Float).
//
// A literal with an exponent is never int or *big.Int, even when its value
// is an integer: encoding/json and (*big.Int).UnmarshalJSON reject it for
// those types. Parsing does not depend on the locale.
func parseNumber(num json.Number) string {
	if _, err := num.Int64(); err == nil {
		return "int"
	} else if strings.Contains(num.String(), ".") || jsonExponentNumber.MatchString(num.String()) {
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
