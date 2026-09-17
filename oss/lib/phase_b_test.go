package lib

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/config"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhaseBReturningParser(t *testing.T) {
	for _, tokens := range [][]string{
		{"ls", "--unknown"}, {"ls", "--marker"}, {"ls", "--marker", "--recursive"},
		{"ls", "--limited-num=invalid"}, {"ls", "--recursive=true"}, {"hash", "--type=invalid"},
	} {
		_, _, err := parseOSSOptions(tokens)
		require.Error(t, err, "%v", tokens)
	}
	for _, tokens := range [][]string{
		{"cp", "--exclude", "*", "--include", "*.txt", "-r", "src", "oss://bucket"},
		{"cp", "--exclude=*", "--include=*.txt", "src", "-r", "oss://bucket"},
	} {
		args, options, err := parseOSSOptions(tokens)
		require.NoError(t, err)
		assert.Equal(t, []string{"cp", "src", "oss://bucket"}, args)
		recursive, err := GetBool(OptionRecursion, options)
		require.NoError(t, err)
		assert.True(t, recursive)
		ok, filters := getFilter(tokens)
		assert.True(t, ok)
		require.Len(t, filters, 2)
		assert.Equal(t, "--exclude", filters[0].name)
		assert.Equal(t, "--include", filters[1].name)
	}
	args, options, err := parseOSSOptions([]string{"hash", "--", "--profile", "-file"})
	require.NoError(t, err)
	assert.Equal(t, []string{"hash", "--profile", "-file"}, args)
	assert.Equal(t, "", *options[OptionRegion].(*string))
	_, options, err = parseOSSOptions([]string{"ls", "--marker=first", "--marker=", "--region", "cn-hangzhou", "--region=cn-beijing"})
	require.NoError(t, err)
	assert.Equal(t, "", *options[OptionMarker].(*string))
	assert.Equal(t, "cn-beijing", *options[OptionRegion].(*string))
	_, options, err = parseOSSOptions([]string{"ls"})
	require.NoError(t, err)
	assert.Equal(t, "", *options[OptionRegion].(*string))
}

func TestPhaseBBridgeConfigAndState(t *testing.T) {
	clearEndpointTestEnv(t)
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fake-ak","access_key_secret":"fake-secret","region_id":"cn-hangzhou","read_timeout":17,"connect_timeout":9,"retry_count":3}]}`), 0600))
	ctx := bridgeTestContext(NewCommandBridge(listCommand.command))
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
	originalRun, originalArgs := parseAndRunCommandImpl, os.Args
	defer func() { parseAndRunCommandImpl = originalRun }()
	sentinel := errors.New("executor failed")
	parseAndRunCommandImpl = func() error {
		assert.NotContains(t, os.Args, "fake-secret")
		assert.Equal(t, "cn-beijing", *bridgeResolvedOptions[OptionRegion].(*string))
		assert.Equal(t, "https://oss-cn-beijing.aliyuncs.com", *bridgeResolvedOptions[OptionEndpoint].(*string))
		assert.Equal(t, "5", *bridgeResolvedOptions[OptionReadTimeout].(*string))
		assert.Equal(t, "9", *bridgeResolvedOptions[OptionConnectTimeout].(*string))
		assert.Equal(t, "2", *bridgeResolvedOptions[OptionRetryTimes].(*string))
		assert.True(t, *bridgeResolvedOptions[OptionSkipVerifyCert].(*bool))
		return sentinel
	}
	err := parseAndRunCommandFromCli(ctx, []string{"oss://bucket", "--region=cn-beijing", "--read-timeout=5", "--retry-count=4", "--retry-times=2", "--skip-secure-verify"}, &listCommand.command)
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, originalArgs, os.Args)
	assert.Nil(t, bridgeResolvedOptions)
	parseAndRunCommandImpl = func() error {
		assert.Equal(t, "cn-hangzhou", *bridgeResolvedOptions[OptionRegion].(*string))
		assert.Equal(t, "17", *bridgeResolvedOptions[OptionReadTimeout].(*string))
		assert.Equal(t, "3", *bridgeResolvedOptions[OptionRetryTimes].(*string))
		assert.NotContains(t, bridgeResolvedOptions, OptionSkipVerifyCert)
		return nil
	}
	require.NoError(t, parseAndRunCommandFromCli(ctx, []string{"oss://bucket"}, &listCommand.command))
}

func TestPhaseBBridgeValidationAndTerminator(t *testing.T) {
	ctx := bridgeTestContext(NewCommandBridge(copyCommand.command))
	for _, args := range [][]string{
		{"src", "oss://bucket", "--jobs=bad"}, {"src", "oss://bucket", "--parallel=0"}, {"src", "oss://bucket", "--include"},
	} {
		err := parseAndRunCommandFromCli(ctx, args, &copyCommand.command)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "credential")
	}
	args := []string{"--profile=selected", "--", "--profile", "--endpoint=x", "--include=*"}
	assert.Equal(t, args[1:], stripCliOnlyFlagsFromArgs(args))
	assert.Equal(t, args, stripOptionWithValue(args, "--endpoint"))
	_, filters := getFilter(args)
	assert.Empty(t, filters)
	ctx = bridgeTestContext(NewCommandBridge(hashCommand.command))
	oldRun := parseAndRunCommandImpl
	defer func() { parseAndRunCommandImpl = oldRun }()
	parseAndRunCommandImpl = func() error {
		assert.Equal(t, []string{"oss", "hash", "--", "--profile"}, os.Args)
		return nil
	}
	require.NoError(t, parseAndRunCommandFromCli(ctx, []string{"--", "--profile"}, &hashCommand.command))
}

func TestPhaseBErrorUnwrap(t *testing.T) {
	cause := oss.ServiceError{Code: "AccessDenied", StatusCode: 403, RequestID: "request"}
	err := CopyError{FileError{ObjectError{BucketError{cause, "bucket"}, "bucket", "key"}, "file"}}
	var serviceErr oss.ServiceError
	require.True(t, errors.As(err, &serviceErr))
	assert.Equal(t, cause, serviceErr)
	assert.ErrorIs(t, err, cause)
}

func TestPhaseBRawCommandRouting(t *testing.T) {
	oldRun := parseAndRunCommandImpl
	defer func() { parseAndRunCommandImpl = oldRun }()
	for _, tokens := range [][]string{
		{"hash", "--type=md5", "help"}, {"hash", "--", "--profile"},
	} {
		called := false
		parseAndRunCommandImpl = func() error {
			called = true
			assert.Equal(t, append([]string{"oss"}, tokens...), os.Args)
			return nil
		}
		root := NewOssCommand()
		root.Execute(bridgeTestContext(root), tokens)
		assert.True(t, called, "%v", tokens)
	}
}

func TestPhaseBHostAliases(t *testing.T) {
	ctx := bridgeTestContext(NewCommandBridge(listCommand.command))
	for short, long := range map[rune]string{'e': "endpoint", 'i': "access-key-id", 'k': "access-key-secret", 't': "sts-token"} {
		require.Same(t, ctx.Flags().Get(long), ctx.Flags().GetByShorthand(short))
	}

	_, _, err := parseBridgeFlags(ctx, []string{"oss://bucket", "--retry-timeout=7", "-p=selected"})
	require.NoError(t, err)
	assert.Equal(t, "7", config.ReadTimeoutFlag(ctx.Flags()).GetStringOrDefault(""))
	assert.Equal(t, []string{"oss://bucket"}, stripCliOnlyFlagsFromArgs([]string{"oss://bucket", "--retry-timeout=7", "-p=selected"}))
}

func TestPhaseBV4RegionAndRetryBudget(t *testing.T) {
	clearEndpointTestEnv(t)
	var requests atomic.Int64
	var correctRegion atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		correctRegion.Store(strings.Contains(r.Header.Get("Authorization"), "/cn-hangzhou/oss/"))
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`<Error><Code>ServiceUnavailable</Code><Message>denied</Message><RequestId>local</RequestId></Error>`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"fake-ak","access_key_secret":"fake-secret","region_id":"cn-hangzhou","retry_count":2}]}`), 0600))
	for _, tc := range []struct {
		flags []string
		count int64
	}{
		{nil, 2}, {[]string{"--retry-count=1"}, 1}, {[]string{"--retry-count=3", "--retry-times=1"}, 1},
	} {
		requests.Store(0)
		ctx := bridgeTestContext(NewCommandBridge(listCommand.command))
		args := []string{"--config-path", path, "--endpoint", server.URL, "--sign-version=v4"}
		err := parseAndRunCommandFromCli(ctx, append(args, tc.flags...), &listCommand.command)
		require.Error(t, err)
		var serviceErr oss.ServiceError
		require.ErrorAs(t, err, &serviceErr)
		assert.Equal(t, "ServiceUnavailable", serviceErr.Code)
		assert.Equal(t, tc.count, requests.Load())
		assert.True(t, correctRegion.Load())
	}
}
