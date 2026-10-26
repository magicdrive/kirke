package commandline

import (
	"fmt"
	"io"
	"os"
)

// PipeReader supplies the JSON piped to kirke.
type PipeReader interface {
	// GetPipeBuffer returns everything on standard input up to EOF. ok is
	// false when standard input is a terminal or a character device (such
	// as /dev/null), that is, when nothing is piped; err reports a failure
	// to read it.
	GetPipeBuffer() (buf string, ok bool, err error)
}

type DefaultPipeReader struct{}

func (DefaultPipeReader) GetPipeBuffer() (string, bool, error) {
	return getPipeBuffer(os.Stdin)
}

var defaultPipeReader PipeReader = DefaultPipeReader{}

func getPipeBuffer(stdin *os.File) (string, bool, error) {
	stat, err := stdin.Stat()
	if err != nil {
		// No usable standard input (e.g. a closed handle): nothing is piped.
		return "", false, nil
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", false, nil
	}
	buf, err := readFromPipe(stdin)
	if err != nil {
		return "", false, err
	}
	return buf, true, nil
}

// readFromPipe reads r to EOF. The input is one JSON document, so it is
// read whole and byte for byte: no line splitting (a last line without a
// line break is kept), no line length limit, CR LF and invalid UTF-8 as is.
func readFromPipe(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	return string(b), nil
}
