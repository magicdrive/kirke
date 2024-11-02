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
	}
	if opt.HelpFlag {
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	jsonStr, err := opt.DecideJSONStr()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	} else if jsonStr == "" {
		if optlength > 0 {
			fmt.Println("No JSON string is provided.")
			os.Exit(1)
		}
		opt.FlagSet.Usage()
		os.Exit(0)
	}

	outputMode, err := opt.DecideOutputMode()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	result, err := core.Apply(jsonStr, opt.RootObjName, outputMode, opt.PointerMode == "on", opt.NullAs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %v\n", err)
		os.Exit(1)
	}

	commandline.GracefulPrintOut(result, opt.PagerMode == "no")
}
