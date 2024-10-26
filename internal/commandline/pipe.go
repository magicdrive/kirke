package commandline

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func GetPipeBuffer() (string, bool) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", false
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		buf, err := readFromPipe()
		if err != nil {
			return "", false
		} else {
			return buf, true
		}
	} else {
		return "", false
	}
}

func readFromPipe() (string, error) {
	var sb strings.Builder
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		sb.WriteString(input)
	}
	return sb.String(), nil
}
