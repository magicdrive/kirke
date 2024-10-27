package commandline

import (
	_ "embed"
	"flag"
)

//go:embed help.txt
var helpMessage string

func OptParse(args []string) (int, *Option, error) {

	optLength := len(args)

	fs := flag.NewFlagSet("kirke", flag.ExitOnError)

	nameOpt := fs.String("name", "AutoGenerated", "Specified root struct name.")
	fs.StringVar(nameOpt, "n", "AutoGenerated", "Specified root struct name.")

	jsonOpt := fs.String("json", "", "Specified json string.")
	fs.StringVar(jsonOpt, "j", "", "Specified json string.")

	inputPathOpt := fs.String("input-file", "", "Specified json string source file.")
	fs.StringVar(inputPathOpt, "f", "", "Specified json string source file.")

	helpFlagOpt := fs.Bool("help", false, "Show this message.")
	fs.BoolVar(helpFlagOpt, "h", false, "Show this message.")

	versionFlagOpt := fs.Bool("version", false, "Show version.")
	fs.BoolVar(versionFlagOpt, "v", false, "Show version.")

	pipeFlagOpt := fs.Bool("pipe", false, "Receive a JSON string from a pipe.")
	fs.BoolVar(helpFlagOpt, "p", false, "Receive a JSON string from a pipe.")

	noPagerFlagOpt := fs.Bool("no-pager", false, "Do not use a pager for output.")

	withPointerOpt := fs.Bool("with-pointer", false, "Make nested struct fields of pointer type.")

	err := fs.Parse(args)
	if err != nil {
		return optLength, nil, err
	}

	result := &Option{
		RootObjName:     *nameOpt,
		Json:            *jsonOpt,
		InputPath:       *inputPathOpt,
		WithPointerFlag: *withPointerOpt,
		HelpFlag:        *helpFlagOpt,
		VersionFlag:     *versionFlagOpt,
		PipeFlag:        *pipeFlagOpt,
		NoPagerFlag:     *noPagerFlagOpt,
		FlagSet:         fs,
	}
	OverRideHelp(fs, result.NoPagerFlag)

	return optLength, result, nil

}

func OverRideHelp(fs *flag.FlagSet, noPagerFlag bool) *flag.FlagSet {
	fs.Usage = func() {
		GracefulPrintOut(helpMessage, noPagerFlag)
	}
	return fs
}
