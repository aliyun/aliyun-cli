// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/internal/legacyapi"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sseRoutingTestConfig = `{
	"current": "default",
	"profiles": [{
		"name": "default",
		"mode": "AK",
		"access_key_id": "test-access-key-id",
		"access_key_secret": "test-access-key-secret",
		"region_id": "cn-shanghai"
	}]
}`

func TestIsStreamingSSEAPI(t *testing.T) {
	sseAPI := &canonicalmeta.API{Name: "GetDasAgentSSE", Protocol: "HTTPS|SSE", Method: "POST"}
	plainAPI := &canonicalmeta.API{Name: "DescribeAuditLogs", Protocol: "HTTPS", Method: "POST"}

	assert.True(t, isStreamingSSEAPI("das", sseAPI))
	assert.True(t, isStreamingSSEAPI("DAS", sseAPI))
	assert.False(t, isStreamingSSEAPI("das", plainAPI), "non-SSE APIs keep the buffered legacy call")
	assert.False(t, isStreamingSSEAPI("ecs", sseAPI), "finishing the SSE wire format for one product must not change others")
	assert.False(t, isStreamingSSEAPI("das", nil))
	assert.False(t, isStreamingSSEAPI("das", &canonicalmeta.API{Name: "GetDasAgentSSE"}))
}

// newDasRoutingTestCommando mirrors the isolated CLI context used by the
// existing PascalCase dry-run tests: unknown flags carry API parameters and the
// profile is read from a throwaway config file.
func newDasRoutingTestCommando(t *testing.T, stdout io.Writer) (*cli.Context, *Commando) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	t.Setenv(plugin.EnvPluginsDir, t.TempDir())
	cleanup := setTestHomeDir(t, t.TempDir())
	t.Cleanup(cleanup)

	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)

	cmd := &cli.Command{}
	cmd.EnableUnknownFlag = true
	AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)
	ctx.Command().Short = &i18n.Text{}

	ctx.Flags().Add(config.NewModeFlag())
	ctx.Flags().Get(config.ModeFlagName).SetAssigned(true)
	ctx.Flags().Get(config.ModeFlagName).SetValue("standard")

	configPath := filepath.Join(t.TempDir(), "config.json")
	assert.NoError(t, os.WriteFile(configPath, []byte(sseRoutingTestConfig), 0600))
	configPathFlag := config.NewConfigurePathFlag()
	configPathFlag.SetAssigned(true)
	configPathFlag.SetValue(configPath)
	ctx.Flags().Add(configPathFlag)

	command := NewCommando(stdout, config.Profile{
		Language:        "en",
		Mode:            "AK",
		AccessKeyId:     "test-access-key-id",
		AccessKeySecret: "test-access-key-secret",
		RegionId:        "cn-shanghai",
	})
	return ctx, command
}

func installDasTestLibrary(t *testing.T, command *Commando, apis ...*canonicalmeta.API) {
	t.Helper()
	fakeRepo := newFakeCanonicalRepo()
	for _, api := range apis {
		fakeRepo.AddAPI("das", "2020-01-16", api)
	}
	command.library = &Library{builtinRepo: getRepository(), canonicalRepo: fakeRepo}
}

func dasSSEAPI() *canonicalmeta.API {
	return canonicalTestAPI(&testLegacyAPI{
		Name:     "GetDasAgentSSE",
		Protocol: "HTTPS|SSE",
		Method:   "POST",
		Parameters: []testLegacyParameter{
			{Name: "Query", Position: "Query", Type: "String", Required: true},
		},
	})
}

func TestMain_DasSSEPascalCaseStreamsOnLegacyChannel(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, dasSSEAPI())

	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("which instances are slow")
	ctx.UnknownFlags().Add(queryFlag)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	called := false
	streamSSE = func(_ *legacyapi.Client, r *legacyapi.Request, emit func(*dara.SSEEvent) error) error {
		called = true
		assert.Equal(t, "GetDasAgentSSE", r.ApiName)
		assert.Equal(t, "RPC", r.GetStyle(), "the SSE API must keep its original call style")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "which instances are slow", r.QueryParams["Query"])
		_ = emit(&dara.SSEEvent{Data: tea.String(`{"delta":"one"}`)})
		_ = emit(&dara.SSEEvent{Data: tea.String(`second`)})
		return nil
	}

	err := command.main(ctx, []string{"das", "GetDasAgentSSE"})
	assert.NoError(t, err)
	assert.True(t, called, "PascalCase SSE APIs must stream through the legacy channel")
	// Same event shape as the kebab-case runtime: one JSON object per event.
	assert.Equal(t, "{\"data\":{\"delta\":\"one\"}}\n{\"data\":\"second\"}\n", stdout.String())
}

func TestMain_DasSSEPascalCaseQuietStillInvokes(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, dasSSEAPI())

	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("q")
	ctx.UnknownFlags().Add(queryFlag)
	QuietFlag(ctx.Flags()).SetAssigned(true)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	called := false
	streamSSE = func(_ *legacyapi.Client, _ *legacyapi.Request, emit func(*dara.SSEEvent) error) error {
		called = true
		return emit(&dara.SSEEvent{Data: tea.String("quiet-event")})
	}

	assert.NoError(t, command.main(ctx, []string{"das", "GetDasAgentSSE"}))
	assert.True(t, called, "--quiet suppresses output, not the call")
	assert.Empty(t, stdout.String())
}

func TestMain_DasSSECarriesLegacyRegionIdFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, dasSSEAPI())

	const region = "cn-north-2-gov-1"
	const endpoint = "das.cn-north-2-gov-1.aliyuncs.com"
	// Keep the routing assertion independent of endpoint changes in the Meta submodule.
	repo, err := meta.MockLoadRepository([]meta.Product{{
		Code: "DAS", Version: "2020-01-16", ApiStyle: "rpc",
		GlobalEndpoint:    "das.aliyuncs.com",
		RegionalEndpoints: map[string]string{region: endpoint},
	}})
	require.NoError(t, err)
	command.library.builtinRepo = repo

	regionFlag := config.NewRegionIdFlag()
	regionFlag.SetAssigned(true)
	regionFlag.SetValue(region)
	ctx.Flags().Add(regionFlag)
	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("q")
	ctx.UnknownFlags().Add(queryFlag)
	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)

	require.NoError(t, command.main(ctx, []string{"das", "GetDasAgentSSE"}))

	var out CliDryRunOutput
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out))
	assert.Equal(t, endpoint, out.Endpoint, "--RegionId must select the regional endpoint")
}

func TestMain_DasNonSSEPascalCaseKeepsLegacyChannel(t *testing.T) {
	stdout := new(bytes.Buffer)
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, canonicalTestAPI(&testLegacyAPI{
		Name:     "DescribeAuditLogs",
		Protocol: "HTTPS",
		Method:   "POST",
	}))

	DryRunJsonFlag(ctx.Flags()).SetAssigned(true)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	streamSSE = func(*legacyapi.Client, *legacyapi.Request, func(*dara.SSEEvent) error) error {
		t.Fatal("non-SSE DAS APIs must not enter the streaming path")
		return nil
	}

	err := command.main(ctx, []string{"das", "DescribeAuditLogs"})
	assert.NoError(t, err)

	var out CliDryRunOutput
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out))
	assert.Equal(t, "RPC", out.Style)
	assert.Equal(t, "DescribeAuditLogs", out.Action)
	assert.NotEmpty(t, out.Endpoint)
}

type flushBuffer struct {
	bytes.Buffer
	flushes int
}

func (w *flushBuffer) Flush() { w.flushes++ }

type failingWriter struct{}

func (w *failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout closed") }

func TestMain_DasSSESkipsEmptyEventsAndFlushes(t *testing.T) {
	stdout := &flushBuffer{}
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, dasSSEAPI())

	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("q")
	ctx.UnknownFlags().Add(queryFlag)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	streamSSE = func(_ *legacyapi.Client, _ *legacyapi.Request, emit func(*dara.SSEEvent) error) error {
		if err := emit(&dara.SSEEvent{}); err != nil {
			return err
		}
		return emit(&dara.SSEEvent{Data: tea.String(`{"delta":"one"}`)})
	}

	assert.NoError(t, command.main(ctx, []string{"das", "GetDasAgentSSE"}))
	assert.Equal(t, "{\"data\":{\"delta\":\"one\"}}\n", stdout.String(), "events without payloads print nothing")
	assert.Equal(t, 1, stdout.flushes, "each printed event is flushed immediately")
}

func TestMain_DasSSEStopsOnWriteError(t *testing.T) {
	stdout := &failingWriter{}
	ctx, command := newDasRoutingTestCommando(t, stdout)
	installDasTestLibrary(t, command, dasSSEAPI())

	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("q")
	ctx.UnknownFlags().Add(queryFlag)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	streamSSE = func(_ *legacyapi.Client, _ *legacyapi.Request, emit func(*dara.SSEEvent) error) error {
		return emit(&dara.SSEEvent{Data: tea.String("event")})
	}

	assert.ErrorContains(t, command.main(ctx, []string{"das", "GetDasAgentSSE"}), "stdout closed")
}

func TestMain_DasSSEPrefersRegionFlagOverRegionId(t *testing.T) {
	ctx, command := newDasRoutingTestCommando(t, new(bytes.Buffer))
	installDasTestLibrary(t, command, dasSSEAPI())

	regionFlag := config.NewRegionFlag()
	regionFlag.SetAssigned(true)
	regionFlag.SetValue("cn-shanghai")
	ctx.Flags().Add(regionFlag)

	regionIDFlag := config.NewRegionIdFlag()
	regionIDFlag.SetAssigned(true)
	regionIDFlag.SetValue("cn-north-2-gov-1")
	ctx.Flags().Add(regionIDFlag)

	queryFlag := &cli.Flag{Name: "Query"}
	queryFlag.SetAssigned(true)
	queryFlag.SetValue("q")
	ctx.UnknownFlags().Add(queryFlag)

	origStream := streamSSE
	defer func() { streamSSE = origStream }()
	var streamed string
	streamSSE = func(_ *legacyapi.Client, r *legacyapi.Request, _ func(*dara.SSEEvent) error) error {
		streamed = r.RegionId
		return nil
	}

	assert.NoError(t, command.main(ctx, []string{"das", "GetDasAgentSSE"}))
	assert.Equal(t, "cn-shanghai", streamed, "legacy precedence is --region > --RegionId")
}
