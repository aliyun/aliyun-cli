package sparksubmit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

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
