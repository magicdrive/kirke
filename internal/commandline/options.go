package commandline

import (
	"encoding/json"
	"errors"
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
	PointerStructMode string
	HelpFlag          bool
	VersionFlag       bool
	ForcePipeFlag     bool
	AutoPagerMode     string
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
			return "", errors.New("your JSON string is corrupted")
		}
	}

	// The piped input; ok is false when nothing is piped.
	pipeF := func() (string, bool, error) {
		if cr.PipeReader == nil {
			cr.PipeReader = defaultPipeReader
		}
		return cr.PipeReader.GetPipeBuffer()
	}

	// --pipe option: the piped input above all else
	if cr.ForcePipeFlag {
		pipeBuf, ok, err := pipeF()
		if err != nil {
			return "", err
		}
		if !ok {
			return "", errors.New("nothing is coming through the pipe")
		}
		return validationF(pipeBuf)
	}

	var jsonStr string

	// --input-file option
	if cr.FilePath != "" {
		if fileBuf, err := common.GetFileContent(cr.FilePath); err == nil {
			jsonStr = fileBuf
		} else {
			return "", fmt.Errorf("could not read file: %w", err)
		}
	}

	// --json option
	if cr.Json != "" {
		jsonStr = cr.Json
	}

	// Neither --json nor --file: the piped input, if any. Standard input is
	// read only here and with --pipe, so that kirke --json and kirke --file
	// neither wait for standard input nor consume it.
	if cr.Json == "" && cr.FilePath == "" {
		pipeBuf, ok, err := pipeF()
		if err != nil {
			return "", err
		}
		if ok {
			jsonStr = pipeBuf
		}
	}

	if jsonStr == "" {
		return "", nil
	}

	return validationF(jsonStr)
}

// DecideOutputMode returns the output mode: --inline or --outline, else
// KIRKE_DEFAULT_OUTPUT_MODE ("inline" or "outline"), else outline. Both
// flags together are an error.
func (cr *Option) DecideOutputMode() (int, error) {
	switch {
	case cr.InlineFlag && cr.OutlineFlag:
		return -1, errors.New("cannot enable --inline and --outline at the same time")
	case cr.InlineFlag:
		return OutputModeInline, nil
	case cr.OutlineFlag:
		return OutputModeOutline, nil
	case cr.DefaultOutputMode == "inline":
		return OutputModeInline, nil
	default:
		return OutputModeOutline, nil
	}
}

func isValidJSON(jsonStr string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}
