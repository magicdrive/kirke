package core

import (
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// jsonExponentInteger matches a JSON number with an exponent part and no
// fraction (RFC 8259, section 6): 1e3, 1E3, -2e+5, 0e0.
var jsonExponentInteger = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)[eE][+-]?[0-9]+$`)

// jsonNumberGrammar is the JSON number grammar (RFC 8259, section 6).
var jsonNumberGrammar = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// parseNumber returns the Go type for a JSON number literal:
//
//   - an integer (no fraction, no exponent) that fits in int64: int;
//   - any other integer: *big.Int;
//   - an exponent and no fraction (1e3, 1E3, -2e+5, 0e0): float64 when
//     strconv.ParseFloat parses it without a range error (underflow to
//     zero or a subnormal included, as encoding/json decodes it), else
//     json.Number (overflow such as 1e400), which json.Unmarshal decodes
//     and json.Marshal writes back as it was;
//   - a fraction, with or without an exponent: float64 when %g of its
//     float64 value prints the literal exactly (1.5, 0.1, 1.23567e+100);
//     otherwise float64 when float64 holds its value as written (1.0,
//     1.2e-4: the shortest decimal form of the float64, which json.Marshal
//     writes, has the literal's value), json.Number when it does not
//     (3.141592653589793238462643383279, 1.5e400, 2.5e-400);
//   - a fraction with an exponent big.Float cannot parse: json.Number when
//     it overflows float64, interface{} otherwise (as before: it underflows
//     to zero, which interface{} decodes);
//   - interface{} when it is not a JSON number (the decoder never
//     produces one).
//
// No literal is typed *big.Float any more: encoding/json cannot decode a
// JSON number into one (it has no UnmarshalJSON), so that type never
// worked. A literal with an exponent is never int or *big.Int, even when
// its value is an integer: encoding/json and (*big.Int).UnmarshalJSON
// reject it for those types. Parsing does not depend on the locale.
func parseNumber(num json.Number) string {
	if jsonExponentInteger.MatchString(num.String()) {
		if _, err := strconv.ParseFloat(num.String(), 64); err != nil {
			// Out of the float64 range.
			return "json.Number"
		}
		return "float64"
	}
	if _, err := num.Int64(); err == nil {
		return "int"
	} else if strings.Contains(num.String(), ".") {
		if !jsonNumberGrammar.MatchString(num.String()) {
			return "interface{}"
		}
		if _, _, parseErr := big.NewFloat(0).Parse(num.String(), 10); parseErr != nil {
			// An exponent too large for big.Float.
			if _, err := strconv.ParseFloat(num.String(), 64); err != nil {
				return "json.Number"
			}
			return "interface{}"
		}
		if float64Exact(num.String()) {
			return "float64"
		}
		return "json.Number"
	} else {
		// Only whether it parses matters: the type is *big.Int either way.
		if _, ok := new(big.Int).SetString(num.String(), 10); !ok {
			//Error converting to big.Int:
			return "interface{}"
		}
		return "*big.Int"
	}
}

// float64Exact reports whether float64 holds the value of the JSON number
// literal s exactly as written: s is in the float64 range and the shortest
// decimal form of its float64 value, which encoding/json writes, has the
// same decimal value as s. Overflow (1e400), underflow (1e-400) and digits
// float64 rounds away (0.30000000000000001) are not.
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
