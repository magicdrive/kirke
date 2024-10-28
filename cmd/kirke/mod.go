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

	jsonStr, err := opt.DecideJSONStr()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	} else if jsonStr == "" {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	if optlength < 1 && jsonStr == "" || opt.HelpFlag {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	if opt.VersionFlag {
		fmt.Printf("kirke version %s\n", version)
		os.Exit(0)
	}

	result, err := core.Apply(jsonStr, opt.RootObjName, opt.WithPointerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	commandline.GracefulPrintOut(result, opt.NoPagerFlag)
}
