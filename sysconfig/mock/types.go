package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvMockEnabled = "ALIBABA_CLOUD_CLI_MOCK"
	EnvMockPath    = "ALIBABA_CLOUD_CLI_MOCK_PATH"
	MockFileName   = "mocks.json"
)

type Record struct {
	Name     string `json:"name"`
	Cmd      string `json:"cmd"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Times    int    `json:"times"`
}

func (r *Record) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	required := []string{"name", "cmd", "exit_code", "stdout", "stderr", "times"}
	for _, field := range required {
		if _, ok := fields[field]; !ok {
			return fmt.Errorf("mock record %s is required", field)
		}
	}

	type record Record
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded record
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*r = Record(decoded)
	return nil
}

func ResolvePath(defaultConfigDir func() string) string {
	if path := os.Getenv(EnvMockPath); path != "" {
		return path
	}
	return filepath.Join(defaultConfigDir(), MockFileName)
}
