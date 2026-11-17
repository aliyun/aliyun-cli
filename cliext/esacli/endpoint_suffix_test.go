package esacli

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
	"github.com/stretchr/testify/require"
)

func TestNPMInstallUsesEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_VERSION", "")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_DOWNLOAD_BASE_URL", "https://assets.oss.aliyuncs.com/releases/")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_NPM_REGISTRY", "https://registry.aliyuncs.com")
	previousGet, previousExec := httpGetFunc, execCommandFunc
	t.Cleanup(func() { httpGetFunc, execCommandFunc = previousGet, previousExec })
	requested := ""
	httpGetFunc = func(address string) (*http.Response, error) {
		requested = address
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(" 1.2.3\n"))}, nil
	}
	ctx, _, _ := newOriginCtx()
	c := NewContext(ctx)
	c.prefixPath, c.npmPath = t.TempDir(), "npm"
	var arguments []string
	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		require.Equal(t, "npm", name)
		arguments = append([]string(nil), args...)
		return exec.Command(os.Args[0], "-test.run=^$")
	}
	require.NoError(t, c.npmInstall(false))
	require.Equal(t, "https://assets.oss.aliyuncs.alibaba/releases/version.txt", requested)
	require.Equal(t, "1.2.3", c.versionRemote)
	require.Contains(t, arguments, npmPackageName+"@1.2.3")
	require.Contains(t, arguments, c.prefixPath)
	require.Contains(t, strings.Join(arguments, " "), "--registry https://registry.aliyuncs.alibaba")
	t.Setenv("ALIBABA_CLOUD_ESA_CLI_NPM_REGISTRY", "https://registry.example.com")
	require.NoError(t, c.npmInstall(false))
	require.Contains(t, strings.Join(arguments, " "), "--registry https://registry.example.com")
}
