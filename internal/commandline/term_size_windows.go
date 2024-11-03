//go:build windows

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
	var csbi struct {
		dwSize              int32
		dwCursorPosition    int32
		wAttributes         uint16
		srWindow            struct{ Left, Top, Right, Bottom int16 }
		dwMaximumWindowSize struct{ X, Y int16 }
	}
	h := syscall.Handle(os.Stdout.Fd())
	ret, _, err := syscall.SyscallN(
		procGetConsoleScreenBufferInfo.Addr(),
		2,
		uintptr(h),
		uintptr(unsafe.Pointer(&csbi)),
		0,
	)
	if ret == 0 {
		return 0, 0, fmt.Errorf("failed to get console size: %v", err)
	}
	height = int(csbi.srWindow.Bottom - csbi.srWindow.Top + 1)
	width = int(csbi.srWindow.Right - csbi.srWindow.Left + 1)
	return
}

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

func ShowWithLess(message string) {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "cmd /c more"
	}

	cmd := exec.Command("cmd", "/c", pagerCmd)
	cmd.Stdin = bytes.NewReader([]byte(message))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		fmt.Print(message)
	}
}
