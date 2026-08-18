// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	runtime "github.com/aliyun/aliyun-openapi-runtime/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func agentEnvelope(t *testing.T, err error, args ...string) cli.AgentErrorEnvelope {
	t.Helper()
	normalized := normalizeAgentError(err, args)
	var agentErr *cli.AgentError
	require.ErrorAs(t, normalized, &agentErr)
	return agentErr.Envelope()
}

func TestNormalizeAgentErrorLocalUsageErrors(t *testing.T) {
	t.Run("kebab unknown flag", func(t *testing.T) {
		cause := &argparser.UnknownFlagError{
			Flag:  "instnace-type",
			Known: []string{"image-id", "instance-type"},
		}
		err := &engine.UsageError{Code: "UNKNOWN_FLAG", Err: fmt.Errorf("%w (help)", cause)}

		envelope := agentEnvelope(t, err, "ecs", "describe-instances")
		assert.Equal(t, cli.UsageErrorCategory, envelope.Category)
		assert.Equal(t, "UNKNOWN_FLAG", envelope.Code)
		assert.Equal(t, cause.Error(), envelope.Message)
		assert.Equal(t, []string{"--instance-type"}, envelope.Suggestions)
		assert.Equal(t, "aliyun ecs describe-instances --help", envelope.Recovery.Command)
		assert.False(t, envelope.Retryable)
	})

	t.Run("missing required parameters", func(t *testing.T) {
		cause := &runtime.MissingRequiredError{Flags: []string{"--image-id", "--instance-type"}}
		err := &engine.UsageError{Code: "MISSING_REQUIRED_PARAMETER", Err: cause}

		envelope := agentEnvelope(t, err, "ecs", "run-instances")
		assert.Equal(t, cli.UsageErrorCategory, envelope.Category)
		assert.Equal(t, "MISSING_REQUIRED_PARAMETER", envelope.Code)
		assert.Equal(t, cause.Error(), envelope.Message)
		assert.Equal(t, []string{"--image-id", "--instance-type"}, envelope.Suggestions)
		assert.Equal(t, "aliyun ecs run-instances --help", envelope.Recovery.Command)
	})

	t.Run("PascalCase invalid parameter", func(t *testing.T) {
		err := &InvalidParameterError{
			Name:              "InstnaceType",
			ProductCode:       "ecs",
			ApiName:           "RunInstances",
			ParameterNames:    []string{"ImageId", "InstanceType"},
			ParameterExamples: map[string]string{"InstanceType": "ecs.g6.large"},
		}

		envelope := agentEnvelope(t, err, "ecs", "RunInstances")
		assert.Equal(t, cli.UsageErrorCategory, envelope.Category)
		assert.Equal(t, "UNKNOWN_FLAG", envelope.Code)
		assert.Equal(t, []string{"--InstanceType"}, envelope.Suggestions)
		assert.NotContains(t, envelope.Suggestions[0], "example")
		assert.Equal(t, "aliyun ecs RunInstances --help", envelope.Recovery.Command)
	})

	t.Run("PascalCase invalid API", func(t *testing.T) {
		err := &InvalidApiError{
			Name: "DescribeInstnaces",
			product: &meta.Product{
				Code:     "ecs",
				ApiNames: []string{"DescribeImages", "DescribeInstances"},
			},
		}

		envelope := agentEnvelope(t, err, "ecs", "DescribeInstnaces")
		assert.Equal(t, cli.UsageErrorCategory, envelope.Category)
		assert.Equal(t, "UNKNOWN_API", envelope.Code)
		assert.Equal(t, []string{"DescribeInstances"}, envelope.Suggestions)
		assert.Equal(t, "aliyun ecs --help", envelope.Recovery.Command)
	})
}

func TestNormalizeAgentErrorCredentialFailure(t *testing.T) {
	cause := errors.New("profile is not configured")
	tests := []error{
		&engine.CredentialError{Err: cause},
		&credentialConfigurationError{Err: cause},
	}

	for _, err := range tests {
		envelope := agentEnvelope(t, err, "ecs", "DescribeInstances")
		assert.Equal(t, cli.AuthenticationErrorCategory, envelope.Category)
		assert.Equal(t, "CREDENTIAL_NOT_CONFIGURED", envelope.Code)
		assert.Equal(t, "aliyun configure", envelope.Recovery.Command)
		assert.False(t, envelope.Retryable)
	}
}

func TestNormalizeAgentErrorOldSDKFailures(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		code      string
		category  cli.AgentErrorCategory
		retryable bool
	}{
		{name: "authentication", status: 401, code: "InvalidAccessKeyId.NotFound", category: cli.AuthenticationErrorCategory},
		{name: "permission", status: 403, code: "Forbidden.RAM", category: cli.PermissionErrorCategory},
		{name: "throttling", status: 429, code: "Throttling.User", category: cli.ThrottlingErrorCategory, retryable: true},
		{name: "service", status: 500, code: "InternalError", category: cli.ServiceErrorCategory, retryable: true},
		{name: "authentication code on 400", status: 400, code: "InvalidSecurityToken.Expired", category: cli.AuthenticationErrorCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"RequestId":"old-%d","Code":%q,"Message":"structured message"}`, tt.status, tt.code)
			err := sdkerrors.NewServerError(tt.status, body, "")
			envelope := agentEnvelope(t, fmt.Errorf("call failed: %w", err), "ecs", "DescribeInstances")

			assert.Equal(t, tt.category, envelope.Category)
			assert.Equal(t, tt.code, envelope.Code)
			assert.Equal(t, fmt.Sprintf("old-%d", tt.status), envelope.RequestID)
			assert.Equal(t, "structured message", envelope.Message)
			assert.Equal(t, tt.retryable, envelope.Retryable)
			assert.Empty(t, envelope.Suggestions)
		})
	}
}

func TestNormalizeAgentErrorTeaSDKFailures(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		code      string
		category  cli.AgentErrorCategory
		retryable bool
	}{
		{name: "authentication", status: 401, code: "Unauthorized", category: cli.AuthenticationErrorCategory},
		{name: "permission", status: 403, code: "AccessDenied", category: cli.PermissionErrorCategory},
		{name: "throttling", status: 429, code: "Throttling.Api", category: cli.ThrottlingErrorCategory, retryable: true},
		{name: "service", status: 503, code: "ServiceUnavailable", category: cli.ServiceErrorCategory, retryable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tea.NewSDKError(map[string]interface{}{
				"statusCode": tt.status,
				"code":       tt.code,
				"message":    "structured tea message",
				"data":       map[string]interface{}{"requestId": fmt.Sprintf("tea-%d", tt.status)},
			})
			envelope := agentEnvelope(t, fmt.Errorf("runtime: %w", err), "ecs", "describe-instances")

			assert.Equal(t, tt.category, envelope.Category)
			assert.Equal(t, tt.code, envelope.Code)
			assert.Equal(t, fmt.Sprintf("tea-%d", tt.status), envelope.RequestID)
			assert.Equal(t, "structured tea message", envelope.Message)
			assert.Equal(t, tt.retryable, envelope.Retryable)
		})
	}
}

func TestNormalizeAgentErrorNetworkAndUnknownFallback(t *testing.T) {
	t.Run("temporary DNS failure", func(t *testing.T) {
		err := &net.DNSError{Err: "temporary failure", Name: "ecs.aliyuncs.com", IsTemporary: true}
		envelope := agentEnvelope(t, err, "ecs", "DescribeInstances")
		assert.Equal(t, cli.NetworkErrorCategory, envelope.Category)
		assert.Equal(t, "NETWORK_FAILURE", envelope.Code)
		assert.True(t, envelope.Retryable)
	})

	t.Run("unknown text is never guessed", func(t *testing.T) {
		err := errors.New("AccessDenied Throttling InvalidAccessKeyId")
		envelope := agentEnvelope(t, err, "ecs", "DescribeInstances")
		assert.Equal(t, cli.InternalErrorCategory, envelope.Category)
		assert.Equal(t, "INTERNAL_ERROR", envelope.Code)
		assert.Equal(t, err.Error(), envelope.Message)
		assert.False(t, envelope.Retryable)
	})
}

func TestAgentErrorEnvelopeEndToEndForBothCommandStyles(t *testing.T) {
	testHome := t.TempDir()
	cleanupHome := setTestHomeDir(t, testHome)
	defer cleanupHome()
	writeMinimalConfigJSON(t, testHome)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"), []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, aimode.Save(filepath.Join(testHome, ".aliyun"), &aimode.AiConfig{Enabled: false}))

	cli.DisableExitCode()
	defer cli.EnableExitCode()

	assertEnvelopeShape := func(t *testing.T, stderr string) map[string]interface{} {
		t.Helper()
		assert.Equal(t, 1, strings.Count(stderr, "\n"), "agent error must be one JSON line")
		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(stderr), &decoded))
		assert.ElementsMatch(t,
			[]string{"ok", "category", "code", "message", "suggestions", "retryable", "requestId", "recovery"},
			mapKeys(decoded))
		return decoded
	}

	t.Run("kebab-case runtime", func(t *testing.T) {
		originalDispatch := runtimeTryDispatch
		runtimeTryDispatch = func(_ *cli.Context, _ []string) (bool, error) {
			unknown := &argparser.UnknownFlagError{Flag: "instnace-type", Known: []string{"image-id", "instance-type"}}
			return true, &engine.UsageError{Code: "UNKNOWN_FLAG", Err: unknown}
		}
		defer func() { runtimeTryDispatch = originalDispatch }()

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
		config.AddFlags(cmd.Flags())
		AddFlags(cmd.Flags())
		commando := NewCommando(stdout, config.Profile{Language: "en"})
		commando.InitWithCommand(cmd)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		originalArgs := os.Args
		os.Args = []string{"aliyun", "ecs", "describe-instances", "--instnace-type", "ecs.g6.large", "--cli-ai-mode"}
		defer func() { os.Args = originalArgs }()
		cmd.Execute(ctx, os.Args[1:])

		assert.Empty(t, stdout.String())
		decoded := assertEnvelopeShape(t, stderr.String())
		assert.Equal(t, "USAGE_ERROR", decoded["category"])
		assert.Equal(t, "UNKNOWN_FLAG", decoded["code"])
		assert.Equal(t, []interface{}{"--instance-type"}, decoded["suggestions"])
	})

	t.Run("PascalCase legacy view", func(t *testing.T) {
		product := meta.Product{
			Code: "Ecs", Version: "2014-05-26", ApiStyle: "rpc", ApiNames: []string{"RunInstances"},
		}
		builtinRepo, err := meta.MockLoadRepository([]meta.Product{product})
		require.NoError(t, err)
		canonicalRepo := newFakeCanonicalRepo()
		canonicalRepo.AddAPI("ecs", "2014-05-26", canonicalTestAPI(&testLegacyAPI{
			Name: "RunInstances", Protocol: "HTTPS", Method: "POST",
			Parameters: []testLegacyParameter{{Name: "InstanceType", Position: "Query", Type: "String"}},
		}))

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
		config.AddFlags(cmd.Flags())
		AddFlags(cmd.Flags())
		commando := NewCommando(stdout, config.Profile{Language: "en"})
		commando.library = &Library{builtinRepo: builtinRepo, canonicalRepo: canonicalRepo}
		commando.InitWithCommand(cmd)
		ctx := cli.NewCommandContext(stdout, stderr)
		ctx.EnterCommand(cmd)

		originalArgs := os.Args
		os.Args = []string{"aliyun", "ecs", "RunInstances", "--InstnaceType", "ecs.g6.large", "--endpoint", "ecs.aliyuncs.com", "--cli-ai-mode"}
		defer func() { os.Args = originalArgs }()
		cmd.Execute(ctx, os.Args[1:])

		assert.Empty(t, stdout.String())
		decoded := assertEnvelopeShape(t, stderr.String())
		assert.Equal(t, "USAGE_ERROR", decoded["category"])
		assert.Equal(t, "UNKNOWN_FLAG", decoded["code"])
		assert.Equal(t, []interface{}{"--InstanceType"}, decoded["suggestions"])
	})
}

func TestAgentErrorEnvelopeForcedOffKeepsHumanOutput(t *testing.T) {
	testHome := t.TempDir()
	cleanupHome := setTestHomeDir(t, testHome)
	defer cleanupHome()
	writeMinimalConfigJSON(t, testHome)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"), []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, aimode.Save(filepath.Join(testHome, ".aliyun"), &aimode.AiConfig{Enabled: true}))

	originalDispatch := runtimeTryDispatch
	runtimeTryDispatch = func(_ *cli.Context, _ []string) (bool, error) {
		unknown := &argparser.UnknownFlagError{Flag: "instnace-type", Known: []string{"instance-type"}}
		return true, &engine.UsageError{Code: "UNKNOWN_FLAG", Err: unknown}
	}
	defer func() { runtimeTryDispatch = originalDispatch }()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(cmd.Flags())
	AddFlags(cmd.Flags())
	commando := NewCommando(stdout, config.Profile{Language: "en"})
	commando.InitWithCommand(cmd)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(cmd)

	cli.DisableExitCode()
	defer cli.EnableExitCode()
	originalArgs := os.Args
	os.Args = []string{"aliyun", "ecs", "describe-instances", "--instnace-type", "ecs.g6.large", "--no-cli-ai-mode"}
	defer func() { os.Args = originalArgs }()
	cmd.Execute(ctx, os.Args[1:])

	assert.Empty(t, stdout.String())
	assert.Contains(t, stderr.String(), "ERROR: unknown flag --instnace-type")
	assert.False(t, strings.HasPrefix(strings.TrimSpace(stderr.String()), "{"))
}

func mapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
