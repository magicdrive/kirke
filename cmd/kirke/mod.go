package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/core"
)

func Excecute(version string) {
	optlength, opt, err := commandline.OptParse(os.Args[1:])
	if err != nil {
		log.Fatalf("Faital Error: %v\n", err)
	}

	if opt.VersionFlag {
		fmt.Printf("kirke version %s\n", version)
		os.Exit(0)
	} else if optlength < 1 || opt.HelpFlag {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	jsonStr, err := opt.DecideJSONStr()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	} else if jsonStr == "" {
		fmt.Printf("No JSON string is provided.\n\n\n\n")
		opt.FlagSet.Usage()
		os.Exit(1)
	}

	outputMode, err := opt.DecideOutputMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	result, err := core.Apply(jsonStr, opt.RootObjName, outputMode, opt.WithPointerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	commandline.GracefulPrintOut(result, opt.NoPagerFlag)
}
