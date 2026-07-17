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

const (
	MaxDelayMs           = 3600000          // 1 hour
	MaxResponseBodySize  = 100 * 1024 * 1024 // 100 MiB
	responseBodyPadByte  = 'x'
)

type Record struct {
	Name             string `json:"name"`
	Cmd              string `json:"cmd"`
	ExitCode         int    `json:"exit_code"`
	Stdout           string `json:"stdout"`
	Stderr           string `json:"stderr"`
	Times            int    `json:"times"`
	DelayMs          int    `json:"delay_ms,omitempty"`
	ResponseBodySize int    `json:"response_body_size,omitempty"`
}

// ExpandToSize returns content truncated or padded to exactly size bytes.
// Padding uses a fixed ASCII byte so mock body length is deterministic for load tests.
func ExpandToSize(content string, size int) string {
	if size <= 0 {
		return content
	}
	if len(content) >= size {
		return content[:size]
	}
	buf := make([]byte, size)
	n := copy(buf, content)
	for i := n; i < size; i++ {
		buf[i] = responseBodyPadByte
	}
	return string(buf)
}

// ResolveStdout returns the mock stdout, expanding to ResponseBodySize when set.
func ResolveStdout(record Record) string {
	if record.ResponseBodySize > 0 {
		return ExpandToSize(record.Stdout, record.ResponseBodySize)
	}
	return record.Stdout
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
