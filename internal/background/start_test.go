package background

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDetachedBackgroundChild(t *testing.T) {
	if os.Getenv("ALIYUN_BACKGROUND_TEST_CHILD") != "1" {
		return
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil || len(input) != 0 {
		os.Exit(2)
	}
	data, err := json.Marshal(os.Args[len(os.Args)-1])
	if err != nil || os.WriteFile(os.Getenv("ALIYUN_BACKGROUND_TEST_RESULT"), data, 0600) != nil {
		os.Exit(3)
	}
	os.Exit(0)
}

func TestStartLaunchesDetachedChildWithArgumentsAndEnvironment(t *testing.T) {
	result := filepath.Join(t.TempDir(), "result.json")
	env := append(os.Environ(), "ALIYUN_BACKGROUND_TEST_CHILD=1", "ALIYUN_BACKGROUND_TEST_RESULT="+result,
		"GORACE=atexit_sleep_ms=0")
	argument := "argument with spaces and 中文"
	require.NoError(t, Start(os.Args[0], []string{"-test.run=^TestDetachedBackgroundChild$", "--", argument}, env))
	var data []byte
	require.Eventually(t, func() bool {
		var err error
		data, err = os.ReadFile(result)
		return err == nil && json.Valid(data)
	}, 10*time.Second, 10*time.Millisecond)
	var got string
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, argument, got)
}

func TestStartReportsMissingExecutable(t *testing.T) {
	require.Error(t, Start(filepath.Join(t.TempDir(), "missing-executable"), nil, os.Environ()))
}
