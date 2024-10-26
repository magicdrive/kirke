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

	// Capture standard output
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }() // Restore after test

	// Create a pipe to capture the output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function
	commandline.GracefulPrintOut(message, true)

	// Close the writer to allow reading all output
	w.Close()

	// Read output from the reader
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Expected output includes a newline
	expected := message
	if buf.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, buf.String())
	}
}

// TestGracefulPrintOut_WithPaging verifies output with pager enabled.
func TestGracefulPrintOut_WithPaging(t *testing.T) {
	message := "Hello, World!\nHello, World!\nHello, World!" // 3 lines
	_ = os.Setenv("PAGER", "cat")                            // Set pager to "cat" for test to capture output

	// Capture standard output
	oldStdout := os.Stdout
	defer func() {
		os.Stdout = oldStdout
		os.Unsetenv("PAGER")
	}() // Restore after test

	// Create a pipe to capture the output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function
	commandline.GracefulPrintOut(message, false)

	// Close the writer to allow reading all output
	w.Close()

	// Read output from the reader
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Expected output includes the entire message
	expected := message
	if buf.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, buf.String())
	}
}
