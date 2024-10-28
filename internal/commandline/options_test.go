package commandline_test

import (
	"os"
	"strings"
	"testing"

	"github.com/magicdrive/kirke/internal/commandline"
)

type MockPipeReader struct {
	Data   string
	Exists bool
}

func (mpr MockPipeReader) GetPipeBuffer() (string, bool) {
	return mpr.Data, mpr.Exists
}

func TestDecideJSONStr(t *testing.T) {
	tests := []struct {
		name           string
		option         commandline.Option
		mockPipeReader commandline.PipeReader
		expectedOutput string
		expectError    bool
	}{
		{
			name: "Valid JSON from pipe with ForcePipeFlag",
			option: commandline.Option{
				ForcePipeFlag: true,
			},
			mockPipeReader: MockPipeReader{Data: `{"name": "test"}`, Exists: true},
			expectedOutput: `{"name": "test"}`,
			expectError:    false,
		},
		{
			name: "Invalid JSON from pipe with ForcePipeFlag",
			option: commandline.Option{
				ForcePipeFlag: true,
			},
			mockPipeReader: MockPipeReader{Data: `{"name": "test"`, Exists: true},
			expectedOutput: "",
			expectError:    true,
		},
		{
			name: "Valid JSON from file with InputPath",
			option: commandline.Option{
				InputPath: "testfile.json",
			},
			expectedOutput: `{"age": 30}`,
			expectError:    false,
		},
		{
			name: "Valid JSON from Json option",
			option: commandline.Option{
				Json: `{"city": "Tokyo"}`,
			},
			expectedOutput: `{"city": "Tokyo"}`,
			expectError:    false,
		},
		{
			name:           "No JSON input",
			option:         commandline.Option{},
			expectedOutput: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.option.PipeReader = tt.mockPipeReader

			if tt.option.InputPath != "" {
				os.WriteFile(tt.option.InputPath, []byte(tt.expectedOutput), 0644)
				defer os.Remove(tt.option.InputPath)
			}

			result, err := tt.option.DecideJSONStr()

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v", tt.expectError, err)
			}

			result = strings.TrimSpace(result)
			if result != tt.expectedOutput {
				t.Errorf("Expected output: %s, got: %s", tt.expectedOutput, result)
			}
		})
	}
}
