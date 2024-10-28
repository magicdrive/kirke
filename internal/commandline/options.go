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
	VersionFlag     bool
	ForcePipeFlag   bool
	NoPagerFlag     bool
	FlagSet         *flag.FlagSet
	PipeReader      PipeReader
}

func (cr *Option) DecideJSONStr() (string, error) {

	validationF := func(jsonStr string) (string, error) {
		if isValidJSON(jsonStr) {
			return jsonStr, nil
		} else {
			return "", fmt.Errorf("Your JSON string is corrupted")
		}
	}

	pipeF := func() (string, error) {
		if cr.PipeReader == nil {
			cr.PipeReader = defaultPipeReader
		}

		if pipeBuf, ok := cr.PipeReader.GetPipeBuffer(); ok {
			return pipeBuf, nil
		} else {
			return "", fmt.Errorf("Nothing is coming through unix pipe.")
		}
	}

	var jsonStr string

	jsonStr, err := pipeF()

	// --pipe option
	if cr.ForcePipeFlag {
		if err != nil {
			return "", err
		} else {
			return validationF(jsonStr)
		}
	}

	// --input-file option
	if cr.InputPath != "" {
		if fileBuf, err := common.GetFileContent(cr.InputPath); err == nil {
			jsonStr = fileBuf
		} else {
			return "", fmt.Errorf("Could not read file: %s", cr.InputPath)
		}
	}

	// --json option
	if cr.Json != "" {
		jsonStr = cr.Json
	}

	if jsonStr == "" {
		return "", nil
	}

	return validationF(jsonStr)

}

func isValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}
