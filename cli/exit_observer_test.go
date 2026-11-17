package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDisabledExitDoesNotNotifyObserver(t *testing.T) {
	previousExitCode, previousObserver := withExitCode, beforeExit
	t.Cleanup(func() {
		withExitCode = previousExitCode
		SetBeforeExit(previousObserver)
	})
	DisableExitCode()
	calls := 0
	SetBeforeExit(func(int) { calls++ })
	Exit(0)
	Exit(7)
	require.Zero(t, calls, "an observer must not record an exit when Exit is disabled")
	SetBeforeExit(nil)
	require.Nil(t, beforeExit, "passing nil must clear the registered observer")
}

func TestExitObserverPreservesProcessResult(t *testing.T) {
	for _, code := range []int{0, 7} {
		for _, clearObserver := range []bool{false, true} {
			t.Run(fmt.Sprintf("code=%d/cleared=%v", code, clearObserver), func(t *testing.T) {
				dir := t.TempDir()
				marker := filepath.Join(dir, "exit-observer")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestExitObserverHelperProcess$")
				cmd.Env = append(os.Environ(),
					"ALIYUN_TEST_EXIT_OBSERVER_HELPER=1",
					"ALIYUN_TEST_EXIT_OBSERVER_CODE="+strconv.Itoa(code),
					"ALIYUN_TEST_EXIT_OBSERVER_MARKER="+marker,
					"ALIYUN_TEST_EXIT_OBSERVER_CLEAR="+strconv.FormatBool(clearObserver),
					"GOCOVERDIR="+dir,
				)
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				require.NoError(t, ctx.Err(), "child timed out")
				if code == 0 {
					require.NoError(t, err)
				} else {
					var exitErr *exec.ExitError
					require.True(t, errors.As(err, &exitErr), "child launch failed: %v", err)
				}
				require.Equal(t, code, cmd.ProcessState.ExitCode())
				require.Equal(t, "business stdout\n", stdout.String())
				require.Equal(t, "business stderr\n", stderr.String())
				if clearObserver {
					require.NoFileExists(t, marker)
				} else {
					data, err := os.ReadFile(marker)
					require.NoError(t, err, "observer must run before os.Exit terminates the child")
					require.Equal(t, fmt.Sprintf("exit=%d\n", code), string(data))
				}
			})
		}
	}
}

func TestExitObserverHelperProcess(t *testing.T) {
	if os.Getenv("ALIYUN_TEST_EXIT_OBSERVER_HELPER") != "1" {
		return
	}
	code, err := strconv.Atoi(os.Getenv("ALIYUN_TEST_EXIT_OBSERVER_CODE"))
	if err != nil {
		os.Exit(98)
	}
	SetBeforeExit(func(code int) {
		if err := os.WriteFile(os.Getenv("ALIYUN_TEST_EXIT_OBSERVER_MARKER"), []byte(fmt.Sprintf("exit=%d\n", code)), 0600); err != nil {
			os.Exit(99)
		}
	})
	if os.Getenv("ALIYUN_TEST_EXIT_OBSERVER_CLEAR") == "true" {
		SetBeforeExit(nil)
	}
	EnableExitCode()
	fmt.Fprintln(os.Stdout, "business stdout")
	fmt.Fprintln(os.Stderr, "business stderr")
	Exit(code)
	os.Exit(97) // Returning from Exit would hide the caller's actual process result.
}
