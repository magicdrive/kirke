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
				FilePath: "testfile.json",
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

			if tt.option.FilePath != "" {
				os.WriteFile(tt.option.FilePath, []byte(tt.expectedOutput), 0644)
				defer os.Remove(tt.option.FilePath)
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

func TestDecideOutputMode(t *testing.T) {
	tests := []struct {
		name         string
		inlineFlag   bool
		outlineFlag  bool
		expectedMode int
		expectError  bool
	}{
		{
			name:         "Inline enabled only",
			inlineFlag:   true,
			outlineFlag:  false,
			expectedMode: commandline.OutputModeInline,
			expectError:  false,
		},
		{
			name:         "Outline enabled only",
			inlineFlag:   false,
			outlineFlag:  true,
			expectedMode: commandline.OutputModeOutline,
			expectError:  false,
		},
		{
			name:         "Both disabled",
			inlineFlag:   false,
			outlineFlag:  false,
			expectedMode: commandline.OutputModeOutline,
			expectError:  false,
		},
		{
			name:         "Both enabled",
			inlineFlag:   true,
			outlineFlag:  true,
			expectedMode: -1,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := commandline.Option{
				InlineFlag:  tt.inlineFlag,
				OutlineFlag: tt.outlineFlag,
			}

			mode, err := cr.DecideOutputMode()
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if mode != tt.expectedMode {
				t.Errorf("Expected mode: %d, got: %d", tt.expectedMode, mode)
			}

			if tt.expectError && err != nil {
				expectedErrorMsg := "cannot enable --inline and --outline at the same time."
				if err.Error() != expectedErrorMsg {
					t.Errorf("Expected error message: %s, got: %s", expectedErrorMsg, err.Error())
				}
			}
		})
	}
}
