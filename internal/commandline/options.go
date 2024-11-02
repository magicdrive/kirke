package commandline

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/magicdrive/kirke/internal/common"
)

const (
	OutputModeInline = iota
	OutputModeOutline
)

type Option struct {
	RootObjName       string
	Json              string
	FilePath          string
	NullAs            string
	WithPointerFlag   bool
	HelpFlag          bool
	VersionFlag       bool
	ForcePipeFlag     bool
	NoPagerFlag       bool
	InlineFlag        bool
	OutlineFlag       bool
	DefaultOutputMode string
	FlagSet           *flag.FlagSet
	PipeReader        PipeReader
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
	if cr.FilePath != "" {
		if fileBuf, err := common.GetFileContent(cr.FilePath); err == nil {
			jsonStr = fileBuf
		} else {
			return "", fmt.Errorf("Could not read file: %s", cr.FilePath)
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

func (cr *Option) DecideOutputMode() (int, error) {
	if cr.InlineFlag == true && cr.OutlineFlag == false {
		return OutputModeInline, nil
	} else if cr.InlineFlag == false && cr.OutlineFlag == true {
		return OutputModeOutline, nil
	} else if cr.InlineFlag == false && cr.OutlineFlag == false {
		if cr.DefaultOutputMode == "inline" {
			return OutputModeInline, nil
		} else if cr.DefaultOutputMode == "outline" {
			return OutputModeOutline, nil
		}
		return OutputModeOutline, nil
	} else if cr.InlineFlag == true && cr.OutlineFlag == true {
		return -1, fmt.Errorf("cannot enable --inline and --outline at the same time.")
	} else {
		return OutputModeOutline, nil
	}
}

func isValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}
