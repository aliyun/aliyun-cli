package sparksubmit

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestSparkEndpointSuffixRunCleansTemporaryConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("spark-submit Run does not support Windows")
	}
	for _, mode := range []string{"success", "failure"} {
		t.Run(mode, func(t *testing.T) {
			c, stdout, _ := sparkLifecycleContext(t)
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			t.Setenv(EnvNoUpdateCheck, "1")
			t.Setenv("ALIYUN_CLI_TEST_SPARK_PROCESS", mode)
			previousLoad, previousLook, previousExec := loadProfileFunc, lookPathFunc, execCommandFunc
			t.Cleanup(func() { loadProfileFunc, lookPathFunc, execCommandFunc = previousLoad, previousLook, previousExec })
			loadProfileFunc = func(*cli.Context) (config.Profile, error) {
				return config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}, nil
			}
			lookPathFunc = func(string) (string, error) { return "java", nil }
			require.NoError(t, os.MkdirAll(filepath.Dir(c.execFilePath), 0755))
			require.NoError(t, os.WriteFile(c.execFilePath, []byte("cached tool"), 0755))
			var temporary string
			execCommandFunc = func(_ string, args ...string) *exec.Cmd {
				temporary = c.runtimeConfDir
				require.NotEmpty(t, temporary)
				props, err := readPropertiesFile(filepath.Join(temporary, "connection.properties"))
				require.NoError(t, err)
				require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.alibaba", props["endpoint"])
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				t.Cleanup(cancel)
				return exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestSparkSubmitExecutionChild$", "--"}, args...)...)
			}
			err := c.Run([]string{"--status", "job-123"})
			if mode == "failure" {
				var status *ExitError
				require.ErrorAs(t, err, &status)
				require.Equal(t, 9, status.ExitCode())
			} else {
				require.NoError(t, err)
				var report sparkExecutionReport
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
				require.Equal(t, temporary, report.ConfDir)
				require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.alibaba", report.Props["endpoint"])
				require.Equal(t, []string{"--status", "job-123"}, report.Arguments)
			}
			require.NoDirExists(t, temporary)
			require.Empty(t, c.runtimeConfDir)
			stored, err := readPropertiesFile(filepath.Join(c.confDir, "connection.properties"))
			require.NoError(t, err)
			require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.com", stored["endpoint"])
		})
	}
}

func TestSparkRejectsInvalidEndpointSuffixBeforeLoadingProfile(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "invalid.example")
	require.ErrorContains(t, NewContext(nil).prepareConnectionProperties(), endpoint.EnvSuffix)
}

func TestSparkEndpointSuffixTemporaryConfigFailures(t *testing.T) {
	for _, failure := range []string{"temporary directory", "copy config", "write stored properties"} {
		t.Run(failure, func(t *testing.T) {
			c, _, _ := sparkLifecycleContext(t)
			t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
			previous := loadProfileFunc
			t.Cleanup(func() { loadProfileFunc = previous })
			loadProfileFunc = func(*cli.Context) (config.Profile, error) {
				return config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}, nil
			}
			require.NoError(t, os.MkdirAll(c.confDir, 0700))
			temporaryRoot := t.TempDir()
			for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(key, temporaryRoot)
			}
			if failure == "temporary directory" {
				blocker := filepath.Join(temporaryRoot, "file")
				require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0600))
				for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
					t.Setenv(key, blocker)
				}
			} else {
				if runtime.GOOS == "windows" {
					t.Skip("requires POSIX permissions or symlinks")
				}
				if failure == "copy config" {
					if os.Geteuid() == 0 {
						t.Skip("root can read directories without read permission")
					}
					unreadable := filepath.Join(c.confDir, "private")
					require.NoError(t, os.Mkdir(unreadable, 0700))
					t.Cleanup(func() { _ = os.Chmod(unreadable, 0700) })
					require.NoError(t, os.Chmod(unreadable, 0))
				} else {
					require.NoError(t, os.Symlink(filepath.Join(c.confDir, "missing", "target"), filepath.Join(c.confDir, "connection.properties")))
				}
			}
			require.Error(t, c.prepareConnectionProperties())
			require.Empty(t, c.runtimeConfDir)
			entries, err := filepath.Glob(filepath.Join(temporaryRoot, "aliyun-spark-conf-*"))
			require.NoError(t, err)
			require.Empty(t, entries, "failed setup must not leave credentials in a temporary directory")
			if failure != "write stored properties" {
				stored, err := readPropertiesFile(filepath.Join(c.confDir, "connection.properties"))
				require.NoError(t, err)
				require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.com", stored["endpoint"])
			}
		})
	}
}

func TestSparkEndpointSuffixUsesTemporaryConfig(t *testing.T) {
	c, _, _ := sparkLifecycleContext(t)
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	previous := loadProfileFunc
	t.Cleanup(func() { loadProfileFunc = previous; os.RemoveAll(c.runtimeConfDir) })
	loadProfileFunc = func(*cli.Context) (config.Profile, error) {
		return config.Profile{Mode: config.AK, AccessKeyId: "id", AccessKeySecret: "secret", RegionId: "cn-hangzhou"}, nil
	}
	require.NoError(t, os.MkdirAll(c.confDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(c.confDir, "spark-defaults.conf"), []byte("spark.test=true"), 0600))
	require.NoError(t, c.prepareConnectionProperties())
	stored, err := readPropertiesFile(filepath.Join(c.confDir, "connection.properties"))
	require.NoError(t, err)
	require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.com", stored["endpoint"])
	require.NotEmpty(t, c.runtimeConfDir)
	effective, err := readPropertiesFile(filepath.Join(c.runtimeConfDir, "connection.properties"))
	require.NoError(t, err)
	require.Equal(t, "emr-serverless-spark.cn-hangzhou.aliyuncs.alibaba", effective["endpoint"])
	require.Equal(t, "id", effective["accessKeyId"])
	require.FileExists(t, filepath.Join(c.runtimeConfDir, "spark-defaults.conf"))
}
