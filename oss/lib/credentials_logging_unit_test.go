package lib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestECSRoleCredentialsLocalRefreshAndCache(t *testing.T) {
	var calls atomic.Int32
	expiry := time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprintf(w, `{"Code":"Success","AccessKeyId":"local-id","AccessKeySecret":"local-secret","SecurityToken":"local-token","Expiration":%q}`, expiry)
	}))
	defer server.Close()
	provider := &EcsRoleAKBuild{url: server.URL}
	for i := 0; i < 2; i++ {
		credentials, err := provider.GetCredentialsE()
		require.NoError(t, err)
		require.Equal(t, "local-id", credentials.GetAccessKeyID())
		require.Equal(t, "local-secret", credentials.GetAccessKeySecret())
		require.Equal(t, "local-token", credentials.GetSecurityToken())
	}
	require.EqualValues(t, 1, calls.Load())
	provider.Expiration = time.Now().Add(-time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	require.Equal(t, "local-id", provider.GetCredentials().GetAccessKeyID())
	require.EqualValues(t, 2, calls.Load())
	for _, tc := range []struct {
		expiry string
		want   bool
	}{{"", false}, {expiry, false}, {"invalid", true}, {"2020-01-01T00:00:00Z", true}} {
		provider.Expiration = tc.expiry
		require.Equal(t, tc.want, provider.IsTimeOut())
	}
}

func TestECSRoleCredentialsRejectMalformedMetadata(t *testing.T) {
	for _, content := range []string{"invalid", `{"Code":"Denied"}`, `{"AccessKeyId":"id"}`, `{"AccessKeyId":"id","AccessKeySecret":"secret","Expiration":"invalid"}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, content) }))
		provider := &EcsRoleAKBuild{url: server.URL}
		credentials, err := provider.GetCredentialsE()
		server.Close()
		require.Error(t, err)
		require.False(t, provider.HasGet)
		require.Empty(t, credentials.GetAccessKeyID())
	}
	provider := &EcsRoleAKBuild{url: "://invalid"}
	_, err := provider.GetCredentialsE()
	require.Error(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		io.WriteString(w, "partial")
	}))
	defer server.Close()
	provider.url = server.URL
	_, err = provider.HttpReqAk()
	require.ErrorContains(t, err, "unexpected EOF")
}

func TestOSSLoggerLevelFilteringAndCleanup(t *testing.T) {
	args, name, level, logger, file := os.Args, logName, logLevel, utilLogger, logFile
	t.Cleanup(func() {
		UnInitLogger()
		os.Args, logName, logLevel, utilLogger, logFile = args, name, level, logger, file
	})
	dir := t.TempDir()
	os.Args = []string{filepath.Join(dir, "ossutil")}
	logFile, utilLogger = nil, nil
	for _, level := range []int{oss.LogOff, oss.Error, oss.Warn, oss.Info, oss.Debug} {
		name := fmt.Sprintf("level-%d.log", level)
		InitLogger(level, name)
		LogError("error-message")
		LogWarn("warn-message")
		LogInfo("info-message")
		LogDebug("debug-message")
		UnInitLogger()
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		for i, message := range []string{"error-message", "warn-message", "info-message", "debug-message"} {
			require.Equal(t, level >= i+1, strings.Contains(string(data), message))
		}
		require.Nil(t, logFile)
		require.Nil(t, utilLogger)
	}
	InitLogger(oss.Info, "missing/logger.log")
	require.Nil(t, utilLogger)
	LogInfo("no initialized logger")
}
