package commandline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

func GetTerminalSize() (height, width int, err error) {
	var ws struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0, fmt.Errorf("syscall error: %v", errno)
	}
	return int(ws.Row), int(ws.Col), nil
}

func CountLines(message string) int {
	return len(strings.Split(message, "\n"))
}

func ShowWithLess(message string) {

	pagerCmd := func() string {
		pager := os.Getenv("PAGER")
		if pager == "" {
			return "less"
		}
		return pager
	}()

	cmd := exec.Command(pagerCmd)
	cmd.Stdin = bytes.NewReader([]byte(message))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		fmt.Println(message)
	}
}

func GracefulPrintOut(message string, noPagerFlag bool) {

	if noPagerFlag {
		fmt.Println(message)
		return
	}

	height, _, err := GetTerminalSize()
	if err != nil {
		//Unable to get terminal size
		fmt.Println(message)
		return
	}

	lines := CountLines(message)

	if lines > height {
		ShowWithLess(message)
	} else {
		fmt.Println(message)
	}
}
