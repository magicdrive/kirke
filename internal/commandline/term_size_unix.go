//go:build !windows

package commandline

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

func ShowWithLess(message string) error {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "less"
	}

	cmd := exec.Command(pagerCmd)
	cmd.Stdin = bytes.NewReader([]byte(message))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// No usable pager: print the message as is.
		_, err := fmt.Print(message)
		return err
	}
	return nil
}
