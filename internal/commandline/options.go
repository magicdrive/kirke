package commandline

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/magicdrive/kirke/internal/common"
)

type Option struct {
	RootObjName     string
	Json            string
	InputPath       string
	WithPointerFlag bool
	HelpFlag        bool
	PipeFlag        bool
	NoPagerFlag     bool
	FlagSet         *flag.FlagSet
}

func (cr *Option) DesideJSONStr() (string, error) {

	var jsonStr string
	jsonStr = cr.Json

	if cr.PipeFlag {
		if pipeBuf, ok := GetPipeBuffer(); ok {
			jsonStr = pipeBuf
		} else {
			return "", fmt.Errorf("Nothing is coming through unix pipe.")
		}
	}

	if cr.InputPath != "" {
		if fileBuf, err := common.GetFileContent(cr.InputPath); err == nil {
			jsonStr = fileBuf
		} else {
			return "", fmt.Errorf("Could not read file: %s", cr.InputPath)
		}
	}

	if isValidJSON(jsonStr) {
		return jsonStr, nil
	} else {
		return "", fmt.Errorf("Your JSON string is corrupted")
	}

}

func isValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}
