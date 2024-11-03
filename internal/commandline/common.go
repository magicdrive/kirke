package commandline

import (
	"fmt"
	"strings"
)

func CountLines(message string) int {
	return len(strings.Split(message, "\n"))
}

func GracefulPrintOut(message string, noPagerFlag bool) {

	if noPagerFlag {
		fmt.Print(message)
		return
	}

	height, _, err := GetTerminalSize()
	if err != nil {
		//Unable to get terminal size
		fmt.Print(message)
		return
	}

	lines := CountLines(message)

	if lines > height {
		ShowWithLess(message)
	} else {
		fmt.Print(message)
	}
}
