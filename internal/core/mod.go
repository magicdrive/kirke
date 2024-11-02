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

	structDef := func() string {
		if OutPutMode == commandline.OutputModeInline {
			return generateInlineStruct(structName, data, nullTypeName)
		} else {
			return generateOutlineStruct(structName, data, withPointer, nullTypeName)
		}
	}()

	formattedCode, err := format.Source([]byte(structDef))
	if err != nil {
		return "", fmt.Errorf("formatting source: %w \n\nSOURCE:\n%s", err, structDef)
	}

	return string(formattedCode), nil
}

