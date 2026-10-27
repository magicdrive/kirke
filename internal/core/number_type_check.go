package core

import (
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// jsonNumberGrammar is the JSON number grammar (RFC 8259, section 6).
var jsonNumberGrammar = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// parseNumber returns the Go type for a JSON number literal. Every type it
// returns for a JSON number decodes that literal with encoding/json
// (json.Unmarshal, no UseNumber needed) and encodes it back to a number of
// the same value:
//
//   - an integer literal (no fraction, no exponent) that fits in int64: int;
//   - any other integer literal: *big.Int (it implements json.Unmarshaler);
//   - a literal with a fraction or an exponent: float64 when float64 holds
//     its value exactly as written, that is when the shortest decimal form of
//     its float64 value (what encoding/json writes) has the same value as
//     the literal (1.5, 0.1, 1.0, 1e3, 1.2e-4, 5e-324); json.Number otherwise,
//     which keeps the literal (3.141592653589793238462643383279, 1e400,
//     1e-400, 12345678901234567890e0);
//   - interface{} for a string that is not a JSON number (the decoder never
//     produces one).
//
// A literal with an exponent is never int or *big.Int, even when its value
// is an integer: encoding/json and (*big.Int).UnmarshalJSON reject it for
// those types. Parsing does not depend on the locale.
func parseNumber(num json.Number) string {
	s := num.String()
	if !jsonNumberGrammar.MatchString(s) {
		return "interface{}"
	}
	if !strings.ContainsAny(s, ".eE") {
		if _, err := strconv.ParseInt(s, 10, 64); err == nil {
			return "int"
		}
		return "*big.Int"
	}
	if float64Exact(s) {
		return "float64"
	}
	return "json.Number"
}

// numberArrayType returns the element type for the numbers of one JSON
// array: the narrowest type of parseNumber that holds every one of them.
// All int: int; all integer literals: *big.Int; all held exactly by
// float64 (integer literals included, so 9007199254740993 is not):
// float64; otherwise json.Number. The numbers must be JSON numbers.
func numberArrayType(nums []json.Number) string {
	allInt, allInteger, allFloat := true, true, true
	for _, n := range nums {
		switch parseNumber(n) {
		case "int":
		case "*big.Int":
			allInt = false
		default:
			allInt, allInteger = false, false
		}
		if allFloat && !float64Exact(n.String()) {
			allFloat = false
		}
	}
	switch {
	case allInt:
		return "int"
	case allInteger:
		return "*big.Int"
	case allFloat:
		return "float64"
	default:
		return "json.Number"
	}
}

// float64Exact reports whether float64 holds the value of the JSON number
// literal s exactly as written: s is in the float64 range and the shortest
// decimal form of its float64 value, which encoding/json writes, has the
// same decimal value as s. Overflow (1e400), underflow (1e-400) and digits
// float64 rounds away (0.30000000000000001, 9007199254740993) are not.
func float64Exact(s string) bool {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return false
	}
	return sameDecimal(s, strconv.FormatFloat(f, 'g', -1, 64))
}

// sameDecimal reports whether the JSON number literals a and b (strconv
// 'g' output is one) have the same decimal value. It compares normalized
// digits and exponents, never the values themselves, so a huge exponent
// (1e-99999999999999999999) costs no more than its length.
func sameDecimal(a, b string) bool {
	an, ad, ae := decimalForm(a)
	bn, bd, be := decimalForm(b)
	if ad == "" || bd == "" {
		return ad == bd // zero, whatever its sign
	}
	return an == bn && ad == bd && ae.Cmp(be) == 0
}

// decimalForm returns the sign, the significant digits (no leading or
// trailing zero) and the exponent e of a JSON number literal, whose value
// is then 0.<digits> × 10^e. Zero has no digits.
func decimalForm(s string) (neg bool, digits string, exp *big.Int) {
	exp = new(big.Int)
	if strings.HasPrefix(s, "-") {
		neg, s = true, s[1:]
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		exp.SetString(strings.TrimPrefix(s[i+1:], "+"), 10)
		s = s[:i]
	}
	intPart, frac, _ := strings.Cut(s, ".")
	digits = intPart + frac
	point := int64(len(intPart))
	trimmed := strings.TrimLeft(digits, "0")
	point -= int64(len(digits) - len(trimmed))
	digits = strings.TrimRight(trimmed, "0")
	if digits == "" {
		return false, "", new(big.Int)
	}
	return neg, digits, exp.Add(exp, big.NewInt(point))
}
