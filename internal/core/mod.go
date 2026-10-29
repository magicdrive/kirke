package core

import (
	"encoding/json"
	"fmt"
	"go/format"
	"strings"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/common"
)

func Apply(jsonStr string, rootObjName string, OutPutMode int, withPointer bool, nullTypeName string) (string, error) {

	var data OrderedMap
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}

	structName := common.ToCamelCase(rootObjName)

	structDef, err := func() (def string, err error) {
		// A key no struct field can decode stops the generation
		// (unrepresentableKey); any other panic is a bug and goes on.
		defer func() {
			if r := recover(); r != nil {
				k, ok := r.(unrepresentableKey)
				if !ok {
					panic(r)
				}
				err = k
			}
		}()
		if OutPutMode == commandline.OutputModeInline {
			return generateInlineStruct(structName, data, nullTypeName), nil
		}
		return generateOutlineStruct(structName, data, withPointer, nullTypeName), nil
	}()
	if err != nil {
		return "", err
	}

	formattedCode, err := format.Source([]byte(structDef))
	if err != nil {
		return "", fmt.Errorf("formatting source: %w \n\nSOURCE:\n%s", err, structDef)
	}

	return string(formattedCode), nil
}
