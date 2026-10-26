package commandline

import (
	"fmt"
	"strings"
)

func CountLines(message string) int {
	return len(strings.Split(message, "\n"))
}

// GracefulPrintOut writes message to standard output, through a pager when
// it is taller than the terminal. It returns the error of a failed write,
// e.g. when the reader of a pipe went away (where SIGPIPE does not end kirke
// first, as on Windows).
func GracefulPrintOut(message string, noPagerFlag bool) error {

	if noPagerFlag {
		_, err := fmt.Print(message)
		return err
	}

	height, _, err := GetTerminalSize()
	if err != nil {
		//Unable to get terminal size
		_, err := fmt.Print(message)
		return err
	}

	lines := CountLines(message)

	if lines > height {
		return ShowWithLess(message)
	}
	_, err = fmt.Print(message)
	return err
}
