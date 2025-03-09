package commandline

import (
	"flag"
	"fmt"
	"os"

	_ "embed"
)

//go:embed help.txt
var helpMessage string

func OptParse(args []string) (int, *Option, error) {

	optLength := len(args)

	fs := flag.NewFlagSet("kirke", flag.ExitOnError)

	// --name
	var defaultName = os.Getenv("KIRKE_DEFAULT_ROOT_NAME")
	if defaultName == "" {
		defaultName = "AutoGenerated"
	}
	nameOpt := fs.String("name", defaultName, "Specified root struct name.")
	fs.StringVar(nameOpt, "n", defaultName, "Specified root struct name.")

	// --json
	jsonOpt := fs.String("json", "", "Specified json string.")
	fs.StringVar(jsonOpt, "j", "", "Specified json string.")

	// --file
	filePathOpt := fs.String("file", "", "Specified json string source file.")
	fs.StringVar(filePathOpt, "f", "", "Specified json string source file.")

	// --null-as
	var defaultNullAs = os.Getenv("KIRKE_DEFAULT_NULL_AS")
	if defaultNullAs == "" {
		defaultNullAs = "interface{}"
	}
	nullAsOpt := fs.String("null-as", defaultNullAs, "Specified type name used to replace nulls from json.")
	fs.StringVar(nullAsOpt, "a", defaultNullAs, "Specified type name used to replace nulls from json.")

	// --help
	helpFlagOpt := fs.Bool("help", false, "Show help message.")
	fs.BoolVar(helpFlagOpt, "h", false, "Show help message.")

	// --version
	versionFlagOpt := fs.Bool("version", false, "Show version.")
	fs.BoolVar(versionFlagOpt, "v", false, "Show version.")

	// --pipe
	forcePpipeFlagOpt := fs.Bool("pipe", false, "Receive a JSON string from a pipe.")
	fs.BoolVar(forcePpipeFlagOpt, "p", false, "Receive a JSON string from a pipe.")

	// pagermode --auto-pager on|off
	var defaultAutoPagerMode = os.Getenv("KIRKE_DEFAULT_AUTO_PAGER_MODE")
	if !(defaultAutoPagerMode == SwitchOn || defaultAutoPagerMode == SwitchOff) {
		defaultAutoPagerMode = SwitchOn
	}
	var autoPagerMode OnOffSwitch = OnOffSwitch(defaultAutoPagerMode)
	fs.Var(&autoPagerMode, "auto-pager", "Specifies whether to use a pager when necessary.")
	fs.Var(&autoPagerMode, "pager", "Specifies whether to use a pager when necessary.")

	// ouputmode --inline --outline
	var defaultOutputMode = os.Getenv("KIRKE_DEFAULT_OUTPUT_MODE")
	inlineFlagOpt := fs.Bool("inline", false, "Create inline struct definition output.")
	fs.BoolVar(inlineFlagOpt, "i", false, "Create inline struct definition output.")
	outlineFlagOpt := fs.Bool("outline", false, "Create outline struct definition output.")
	fs.BoolVar(outlineFlagOpt, "o", false, "Create outline struct definition output.")

	// pointer mode --pointer-struct on|off
	var defaultPointerStructMode = os.Getenv("KIRKE_DEFAULT_POINTER_STRUCT_MODE")
	if !(defaultPointerStructMode == SwitchOn || defaultPointerStructMode == SwitchOff) {
		defaultPointerStructMode = SwitchOff
	}
	var pointerStructMode OnOffSwitch = OnOffSwitch(defaultPointerStructMode)
	fs.Var(&pointerStructMode, "pointer-struct", "Make nested struct fields of pointer type.")
	fs.Var(&pointerStructMode, "pointer", "Make nested struct fields of pointer type.")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "\nHelpOption:")
		fmt.Fprintln(os.Stderr, "    kirke --help")
	}
	err := fs.Parse(args)
	if err != nil {
		return optLength, nil, err
	}

	result := &Option{
		RootObjName:       *nameOpt,
		Json:              *jsonOpt,
		FilePath:          *filePathOpt,
		NullAs:            *nullAsOpt,
		PointerStructMode: pointerStructMode.String(),
		HelpFlag:          *helpFlagOpt,
		VersionFlag:       *versionFlagOpt,
		ForcePipeFlag:     *forcePpipeFlagOpt,
		AutoPagerMode:     autoPagerMode.String(),
		InlineFlag:        *inlineFlagOpt,
		OutlineFlag:       *outlineFlagOpt,
		DefaultOutputMode: defaultOutputMode,
		FlagSet:           fs,
	}
	OverRideHelp(fs, result.AutoPagerMode == SwitchOff)

	return optLength, result, nil
}

func OverRideHelp(fs *flag.FlagSet, noPagerFlag bool) *flag.FlagSet {
	fs.Usage = func() {
		GracefulPrintOut(helpMessage, noPagerFlag)
	}
	return fs
}
