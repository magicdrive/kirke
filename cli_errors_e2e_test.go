package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLI_ErrorMessages pins what kirke prints on standard error, and its
// exit status, for each user error.
func TestCLI_ErrorMessages(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantPrefix string
		wantExact  bool
	}{
		{"corrupt --json", []string{"-j", `{"a":`}, "", "Error: your JSON string is corrupted\n", true},
		{"corrupt --pipe", []string{"--pipe"}, `{"a":`, "Error: your JSON string is corrupted\n", true},
		{"--inline and --outline", []string{"-j", `{"a":1}`, "--inline", "--outline"}, "", "Error: cannot enable --inline and --outline at the same time\n", true},
		{"missing --file", []string{"-f", missing}, "", "Error: could not read file: open " + missing + ": ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runKirke(t, []byte(tt.stdin), append(tt.args, "--auto-pager", "off")...)
			var ee *exec.ExitError
			if !errors.As(r.err, &ee) || ee.ExitCode() != 1 {
				t.Errorf("want exit status 1, got %v", r.err)
			}
			if tt.wantExact && r.stderr != tt.wantPrefix || !strings.HasPrefix(r.stderr, tt.wantPrefix) {
				t.Errorf("stderr %q, want %q", r.stderr, tt.wantPrefix)
			}
			if r.stdout != "" {
				t.Errorf("unexpected output %q", r.stdout)
			}
		})
	}

	t.Run("--pipe with nothing piped", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("NUL is not reported as a character device by every Windows version")
		}
		devnull, err := os.Open(os.DevNull)
		if err != nil {
			t.Fatal(err)
		}
		defer devnull.Close()
		cmd := exec.Command(pipeTestBinary(t), "--pipe", "--auto-pager", "off")
		cmd.Env = kirkeTestEnv()
		cmd.Stdin = devnull
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err = cmd.Run()
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			t.Errorf("want exit status 1, got %v", err)
		}
		if got := stderr.String(); got != "Error: nothing is coming through the pipe\n" {
			t.Errorf("stderr %q", got)
		}
	})
}
