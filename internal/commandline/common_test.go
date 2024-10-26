package commandline_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
)

// TestCountLines verifies line count functionality.
func TestCountLines(t *testing.T) {
	message := "Line1\nLine2\nLine3"
	expected := 3
	if count := commandline.CountLines(message); count != expected {
		t.Errorf("Expected %d lines, got %d", expected, count)
	}
}

// TestGracefulPrintOut_NoPagerFlag verifies output when noPagerFlag is enabled.
func TestGracefulPrintOut_NoPagerFlag(t *testing.T) {
	message := "Hello, World!"

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }() // Restore after test

	r, w, _ := os.Pipe()
	os.Stdout = w

	commandline.GracefulPrintOut(message, true)

	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	expected := message
	if buf.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, buf.String())
	}
}

// TestGracefulPrintOut_WithPaging verifies output with pager enabled.
func TestGracefulPrintOut_WithPaging(t *testing.T) {
	message := "Hello, World!\nHello, World!\nHello, World!"
	_ = os.Setenv("PAGER", "cat")

	oldStdout := os.Stdout
	defer func() {
		os.Stdout = oldStdout
		os.Unsetenv("PAGER")
	}()

	r, w, _ := os.Pipe()
	os.Stdout = w

	commandline.GracefulPrintOut(message, false)

	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	expected := message
	if buf.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, buf.String())
	}
}
