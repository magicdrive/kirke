package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

func parseNumber(num json.Number) string {
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
		intValue := new(big.Int)
		intValue, ok := intValue.SetString(num.String(), 10)
		if !ok {
			//Error converting to big.Int:
			return "interface{}"
		}
		return "*big.Int"
	}
}
