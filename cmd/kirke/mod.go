package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/magicdrive/kirke/internal/commandline"
	"github.com/magicdrive/kirke/internal/core"
)

func Excecute() {
	optlength, opt, err := commandline.OptParse(os.Args[1:])
	if err != nil {
		log.Fatalf("Faital Error: %v\n", err)
	}

	if optlength < 1 || opt.HelpFlag {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	jsonStr, err := opt.DesideJSONStr()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
	} else if jsonStr == "" {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	result, err := core.Apply(jsonStr, opt.RootObjName, opt.WithPointerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
	}

	commandline.GracefulPrintOut(result, opt.NoPagerFlag)
}
