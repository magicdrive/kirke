package commandline

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadFromPipe(t *testing.T) {
	long := strings.Repeat("a", 200*1024) // beyond bufio.Scanner's 64 KiB token limit
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"one line without newline", `{"a":1}`},
		{"one line with newline", "{\"a\":1}\n"},
		{"multiple lines, last without newline", "{\n\"a\": 1,\n\"b\": 2}"},
		{"multiple lines with newline", "{\n\"a\": 1,\n\"b\": 2}\n"},
		{"CRLF", "{\r\n\"a\": 1\r\n}\r\n"},
		{"CRLF, last line without CRLF", "{\r\n\"a\": 1\r\n}"},
		{"lone CR", "{\r\"a\": 1\r}"},
		{"Unicode", "{\"名前\":\"キルケー 🦉\"}"},
		{"invalid UTF-8", "{\"a\":\"\xff\xfe\xc3\"}"},
		{"NUL bytes", "{\"a\":\"\x00\"}\x00"},
		{"only newlines", "\n\n\n"},
		{"long line without newline", `{"a":"` + long + `"}`},
		{"long line with newline", `{"a":"` + long + "\"}\n"},
	}
	readers := map[string]func(string) io.Reader{
		"whole":      func(s string) io.Reader { return strings.NewReader(s) },
		"one byte":   func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) },
		"data+EOF":   func(s string) io.Reader { return iotest.DataErrReader(strings.NewReader(s)) },
		"half reads": func(s string) io.Reader { return iotest.HalfReader(strings.NewReader(s)) },
	}
	for _, tt := range tests {
		for rname, mk := range readers {
			t.Run(tt.name+"/"+rname, func(t *testing.T) {
				got, err := readFromPipe(mk(tt.input))
				if err != nil {
					t.Fatalf("readFromPipe: %v", err)
				}
				if got != tt.input {
					t.Errorf("readFromPipe changed the input:\n got %d bytes %.80q\nwant %d bytes %.80q", len(got), got, len(tt.input), tt.input)
				}
			})
		}
	}
}

func TestReadFromPipe_Error(t *testing.T) {
	errBoom := errors.New("boom")
	for name, r := range map[string]io.Reader{
		"error at once":        iotest.ErrReader(errBoom),
		"error after data":     io.MultiReader(strings.NewReader("{\"a\":"), iotest.ErrReader(errBoom)),
		"timeout on 2nd read":  iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("{}"))),
		"closed pipe":          iotest.ErrReader(io.ErrClosedPipe),
		"cancelled (EOF-less)": iotest.ErrReader(os.ErrDeadlineExceeded),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := readFromPipe(r)
			if err == nil {
				t.Fatalf("readFromPipe returned %q and no error", got)
			}
			if !strings.HasPrefix(err.Error(), "read standard input: ") {
				t.Errorf("error %q does not say what failed", err)
			}
			if errors.Unwrap(err) == nil {
				t.Errorf("error %q does not wrap its cause", err)
			}
			if got != "" {
				t.Errorf("partial input %q returned with the error", got)
			}
		})
	}
}

func TestGetPipeBuffer(t *testing.T) {
	dir := t.TempDir()

	// A regular file (kirke < file.json) and a pipe are read; the input is
	// returned exactly, last line without newline included.
	file := filepath.Join(dir, "in.json")
	if err := os.WriteFile(file, []byte("{\r\n\"a\": 1\r\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	buf, ok, err := getPipeBuffer(f)
	if err != nil || !ok || buf != "{\r\n\"a\": 1\r\n}" {
		t.Errorf("file: got %q, %v, %v", buf, ok, err)
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	want := bytes.Repeat([]byte(`{"k":"v"}`), 1)
	go func() {
		pw.Write(want)
		pw.Close()
	}()
	buf, ok, err = getPipeBuffer(pr)
	if err != nil || !ok || buf != string(want) {
		t.Errorf("pipe: got %q, %v, %v", buf, ok, err)
	}

	// A file that cannot be stat'ed (closed) is no input, not an error.
	closed, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	closed.Close()
	if buf, ok, err := getPipeBuffer(closed); ok || err != nil || buf != "" {
		t.Errorf("closed: got %q, %v, %v", buf, ok, err)
	}
}

func TestGetPipeBuffer_CharDevice(t *testing.T) {
	// A character device (a terminal; /dev/null) means nothing is piped:
	// it is not read.
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if st, err := f.Stat(); err != nil || st.Mode()&os.ModeCharDevice == 0 {
		t.Skipf("%s is not a character device here", os.DevNull)
	}
	if buf, ok, err := getPipeBuffer(f); ok || err != nil || buf != "" {
		t.Errorf("got %q, %v, %v", buf, ok, err)
	}
}
