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

func requireAgentEnvelope(t *testing.T, err error, args []string, validator RecoverySearchValidator) cli.AgentErrorEnvelope {
	t.Helper()
	normalized := normalizeAgentErrorWithSearch(err, args, validator)
	var agentErr *cli.AgentError
	require.ErrorAs(t, normalized, &agentErr)
	assert.ErrorIs(t, agentErr, err)
	return agentErr.Envelope()
}

func TestNormalizeAgentErrorSupportedLocalRecoveries(t *testing.T) {
	t.Run("unknown product uses validated product search", func(t *testing.T) {
		repo, err := meta.MockLoadRepository([]meta.Product{{Code: "Ecs"}})
		require.NoError(t, err)
		cause := &InvalidProductError{Code: "ecx", library: &Library{builtinRepo: repo}}
		var request RecoverySearchRequest
		envelope := requireAgentEnvelope(t, cause, []string{"ecx"}, func(got RecoverySearchRequest) bool {
			request = got
			return true
		})

		assert.Equal(t, `"ecx" is not a valid command or product.`, envelope.Message)
		assert.Equal(t, []string{"ecs"}, envelope.DidYouMean)
		assert.Equal(t, "search_product", envelope.Recovery.Action)
		assert.Equal(t, "aliyun --help-search ecs", envelope.Recovery.Command)
		assert.Equal(t, "Search products related to ecs.", envelope.Recovery.Hint)
		assert.Equal(t, RecoverySearchRequest{Keyword: "ecs"}, request)
	})

	t.Run("unknown API derives a resource keyword from a real candidate", func(t *testing.T) {
		cause := &InvalidApiError{
			Name: "DescribeInstnaces",
			product: &meta.Product{
				Code: "ecs", ApiNames: []string{"DescribeImages", "DescribeInstances"},
			},
		}
		var requests []RecoverySearchRequest
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstnaces"}, func(got RecoverySearchRequest) bool {
			requests = append(requests, got)
			return got.Keyword == "Instances"
		})

		assert.Equal(t, `"DescribeInstnaces" is not a valid api.`, envelope.Message)
		assert.Equal(t, []string{"DescribeInstances", "describe-instances"}, envelope.DidYouMean)
		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs --help-search Instances", envelope.Recovery.Command)
		assert.Equal(t, "Search APIs related to Instances.", envelope.Recovery.Hint)
		assert.Equal(t, []RecoverySearchRequest{
			{Product: "ecs", Style: "pascal", Keyword: "Instances"},
		}, requests)
	})

	t.Run("legacy unknown parameter message excludes Help recovery text", func(t *testing.T) {
		cause := &InvalidParameterError{
			Name: "InstnaceType", ProductCode: "ecs", ApiName: "RunInstances", ParameterNames: []string{"InstanceType"},
		}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "RunInstances"}, func(RecoverySearchRequest) bool { return true })

		assert.Equal(t, `"--InstnaceType" is not a valid parameter or flag.`, envelope.Message)
		assert.NotContains(t, envelope.Message, "aliyun help")
	})

	t.Run("unknown CLI subcommand points at its current parent", func(t *testing.T) {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		parent := &cli.Command{Name: "configure"}
		ctx.EnterCommand(parent)
		cause := cli.NewInvalidCommandError("profiel", ctx)

		envelope := requireAgentEnvelope(t, cause, []string{"configure", "profiel"}, nil)
		assert.Equal(t, "inspect_parent_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun configure --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect commands under the current parent.", envelope.Recovery.Hint)
	})

	t.Run("unknown CLI subcommand publishes only validated current-level search", func(t *testing.T) {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		parent := &cli.Command{Name: "configure"}
		parent.AddSubCommand(&cli.Command{Name: "profile"})
		ctx.EnterCommand(parent)
		cause := cli.NewInvalidCommandError("profiel", ctx)

		envelope := requireAgentEnvelope(t, cause, []string{"configure", "profiel"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, RecoverySearchRequest{Product: "configure", Keyword: "profile"}, request)
			return true
		})
		assert.Equal(t, "search_command", envelope.Recovery.Action)
		assert.Equal(t, "aliyun configure --help-search profile", envelope.Recovery.Command)
		assert.Equal(t, "Search commands under the current parent related to profile.", envelope.Recovery.Hint)
	})

	t.Run("unknown host flag publishes only validated current-level search", func(t *testing.T) {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		command := &cli.Command{Name: "configure"}
		command.Flags().Add(&cli.Flag{Name: "region"})
		ctx.EnterCommand(command)
		cause := cli.NewInvalidFlagError("regoin", ctx)

		envelope := requireAgentEnvelope(t, cause, []string{"configure", "--regoin"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, RecoverySearchRequest{Product: "configure", Keyword: "region"}, request)
			return true
		})
		assert.Equal(t, []string{"--region"}, envelope.DidYouMean)
		assert.Equal(t, "search_parameter", envelope.Recovery.Action)
		assert.Equal(t, "aliyun configure --help-search region", envelope.Recovery.Command)
		assert.Equal(t, "Search flags for this command related to region.", envelope.Recovery.Hint)
	})

	t.Run("unknown kebab flag searches a validated request parameter", func(t *testing.T) {
		cause := &engine.UsageError{Code: "UNKNOWN_FLAG", Err: &argparser.UnknownFlagError{
			Flag: "instnace-type", Known: []string{"image-id", "instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, RecoverySearchRequest{
				Product: "ecs", API: "describe-instances", Section: "request", Style: "kebab", Keyword: "instance-type",
			}, request)
			return true
		})

		assert.Equal(t, []string{"--instance-type"}, envelope.DidYouMean)
		assert.Equal(t, "search_parameter", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs describe-instances --help-search instance-type", envelope.Recovery.Command)
		assert.Equal(t, "Search request parameters related to instance-type.", envelope.Recovery.Hint)
	})

	t.Run("missing required parameter uses complete request help", func(t *testing.T) {
		cause := &engine.UsageError{Code: "MISSING_REQUIRED_PARAMETER", Err: &runtime.MissingRequiredError{
			Flags: []string{"--image-id", "--instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, nil)

		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun help ecs run-instances --cli-section request", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the complete request help and provide every required parameter.", envelope.Recovery.Hint)
	})

	t.Run("legacy missing required parameter uses complete request help", func(t *testing.T) {
		cause := cli.NewErrorWithTip(&LegacyMissingRequiredError{
			Err: errors.New("required parameters not assigned: --InstanceId"),
		}, "use `aliyun ecs DescribeInstanceAttribute --help` to get more information")
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstanceAttribute"}, nil)

		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "required parameters not assigned: --InstanceId", envelope.Message)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun help ecs DescribeInstanceAttribute --cli-section request", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the complete request help and provide every required parameter.", envelope.Recovery.Hint)
	})

	t.Run("invalid argument preserves typed parameter context", func(t *testing.T) {
		text := "--tags: invalid JSON: unexpected end of JSON input"
		cause := &engine.UsageError{Code: "INVALID_ARGUMENT", Err: &argparser.InvalidArgumentError{
			Parameter: "tags", Flag: "--tags", FieldPath: "Tags", ExpectedType: "array", Err: errors.New(text),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, "tags", request.Keyword)
			return true
		})

		assert.Equal(t, text, envelope.Message)
		assert.Equal(t, "inspect_parameter_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs run-instances --tags --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect help for --tags and correct its syntax or type.", envelope.Recovery.Hint)
	})

	t.Run("invalid argument without a real top-level flag uses validated request search", func(t *testing.T) {
		text := "Tags.0.Value must be a string"
		cause := &engine.UsageError{Code: "INVALID_ARGUMENT", Err: &argparser.InvalidArgumentError{
			FieldPath: "Tags.0.Value", ExpectedType: "string", Err: errors.New(text),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, RecoverySearchRequest{
				Product: "ecs", API: "run-instances", Section: "request", Style: "kebab", Keyword: "Tags.0.Value",
			}, request)
			return true
		})

		assert.Equal(t, text, envelope.Message)
		assert.Equal(t, "search_parameter", envelope.Recovery.Action)
		assert.Equal(t, "aliyun help ecs run-instances --cli-section request --help-search Tags.0.Value", envelope.Recovery.Command)
	})

	t.Run("invalid option combination identifies options to remove", func(t *testing.T) {
		text := "--cli-dry-run-json cannot be used with --pager"
		cause := &engine.UsageError{Code: "INVALID_OPTION_COMBINATION", Err: &engine.InvalidOptionCombinationError{
			Options: []string{"--cli-dry-run-json", "--pager"}, Err: errors.New(text),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, nil)

		assert.Equal(t, text, envelope.Message)
		assert.Equal(t, "fix_option_combination", envelope.Recovery.Action)
		assert.Empty(t, envelope.Recovery.Command)
		assert.Equal(t, "Remove one of the conflicting options: --cli-dry-run-json, --pager.", envelope.Recovery.Hint)
	})

	t.Run("invalid header searches validated header usage without copying its value", func(t *testing.T) {
		cause := &engine.UsageError{Code: "INVALID_HEADER", Err: &engine.InvalidHeaderError{
			Input: "Authorization=secret-value=still-secret", ExpectedFormat: "Name=Value",
			Err: errors.New(`invalid header format "broken", expected Name=Value`),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, "header", request.Keyword)
			return true
		})

		assert.Equal(t, "inspect_header_usage", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs describe-instances --header --help", envelope.Recovery.Command)
		assert.NotContains(t, envelope.Recovery.Command, "secret-value")
		assert.Equal(t, "Inspect header usage and pass each header as Name=Value.", envelope.Recovery.Hint)
	})

	t.Run("unreadable body file searches validated body-file usage without copying the path", func(t *testing.T) {
		cause := &engine.UsageError{Code: "INVALID_BODY_FILE", Err: &engine.InvalidBodyFileError{
			Path: "/private/secret/request.json", Err: errors.New("--body-file: permission denied"),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, "body-file", request.Keyword)
			return true
		})

		assert.Equal(t, "fix_body_file", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs run-instances --body-file --help", envelope.Recovery.Command)
		assert.NotContains(t, envelope.Recovery.Command, "/private/secret")
		assert.Equal(t, "Check that --body-file points to a readable file.", envelope.Recovery.Hint)
	})
}

func TestNormalizeAgentErrorSearchValidationFallbackAndCommandStyle(t *testing.T) {
	t.Run("baseline unknown API tokenizes the invalid name and uses a validated keyword", func(t *testing.T) {
		cause := &InvalidBaselineCommandError{
			Product: "bssopenapi",
			Command: "QueryMonthlyBill",
			Err:     errors.New(`"QueryMonthlyBill" is not a valid api.`),
		}
		var requests []RecoverySearchRequest
		envelope := requireAgentEnvelope(t, cause, []string{"bssopenapi", "QueryMonthlyBill"}, func(got RecoverySearchRequest) bool {
			requests = append(requests, got)
			return got.Keyword == "Bill"
		})

		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "aliyun bssopenapi --help-search Bill", envelope.Recovery.Command)
		assert.Equal(t, "Search APIs related to Bill.", envelope.Recovery.Hint)
		assert.Contains(t, requests, RecoverySearchRequest{Product: "bssopenapi", Style: "pascal", Keyword: "MonthlyBill"})
		assert.Contains(t, requests, RecoverySearchRequest{Product: "bssopenapi", Style: "pascal", Keyword: "Bill"})
	})

	t.Run("failed API search validation falls back to product inspection", func(t *testing.T) {
		cause := &InvalidApiError{
			Name:    "GetCouponLits",
			product: &meta.Product{Code: "billing", ApiNames: []string{"GetCouponList"}},
		}
		envelope := requireAgentEnvelope(t, cause, []string{"billing", "GetCouponLits"}, func(RecoverySearchRequest) bool { return false })

		assert.Equal(t, "inspect_product_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun billing --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the available APIs for this product.", envelope.Recovery.Hint)
	})

	t.Run("failed product search validation falls back to root inspection", func(t *testing.T) {
		cause := &InvalidProductError{Code: "not-a-product"}
		envelope := requireAgentEnvelope(t, cause, []string{"not-a-product"}, func(RecoverySearchRequest) bool { return false })

		assert.Equal(t, "inspect_root_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the available products.", envelope.Recovery.Hint)
	})

	t.Run("failed parameter search validation falls back to request inspection", func(t *testing.T) {
		cause := &engine.UsageError{Code: "UNKNOWN_FLAG", Err: &argparser.UnknownFlagError{
			Flag: "not-a-parameter", Known: []string{"image-id", "instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, func(RecoverySearchRequest) bool { return false })

		assert.Equal(t, "inspect_action_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs describe-instances --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the action help and correct the parameter or flag.", envelope.Recovery.Hint)
	})

	t.Run("PascalCase recovery keeps style and explicit version", func(t *testing.T) {
		cause := &InvalidParameterError{
			Name: "InstnaceType", ProductCode: "ecs", ApiName: "RunInstances", ParameterNames: []string{"InstanceType"},
		}
		var request RecoverySearchRequest
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "RunInstances", "--version", "2014-05-26"},
			func(got RecoverySearchRequest) bool { request = got; return true })

		assert.Equal(t, "aliyun ecs RunInstances --version 2014-05-26 --help-search InstanceType", envelope.Recovery.Command)
		assert.Equal(t, "pascal", request.Style)
		assert.Equal(t, "2014-05-26", request.Version)
	})

	t.Run("known PascalCase parameter syntax error uses versioned L3 help", func(t *testing.T) {
		cause := &argparser.InvalidArgumentError{
			Flag: "--InstanceIds", Parameter: "InstanceIds", FieldPath: "InstanceIds",
			Err: errors.New("--InstanceIds: invalid JSON"),
		}
		validatorCalled := false
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "DescribeInstances", "--version=2014-05-26"},
			func(RecoverySearchRequest) bool { validatorCalled = true; return true })

		assert.Equal(t, "inspect_parameter_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeInstances --version 2014-05-26 --InstanceIds --help", envelope.Recovery.Command)
		assert.False(t, validatorCalled, "known top-level flags do not need Search validation")
	})

	t.Run("kebab recovery keeps style and explicit API version", func(t *testing.T) {
		cause := &engine.UsageError{Code: "UNKNOWN_FLAG", Err: &argparser.UnknownFlagError{
			Flag: "instnace-type", Known: []string{"instance-type"},
		}}
		var request RecoverySearchRequest
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "run-instances", "--api-version", "2014-05-26"},
			func(got RecoverySearchRequest) bool { request = got; return true })

		assert.Equal(t, "aliyun ecs run-instances --api-version 2014-05-26 --help-search instance-type", envelope.Recovery.Command)
		assert.Equal(t, "kebab", request.Style)
		assert.Equal(t, "2014-05-26", request.Version)
	})

	t.Run("required recovery keeps prefix Section form and explicit API version", func(t *testing.T) {
		cause := &runtime.MissingRequiredError{Flags: []string{"--instance-id"}}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "describe-instance-attribute", "--api-version=2014-05-26"}, nil)

		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun help ecs describe-instance-attribute --api-version 2014-05-26 --cli-section request", envelope.Recovery.Command)
	})
}

func TestNormalizeAgentErrorStrictlyBypassesExcludedErrors(t *testing.T) {
	serverBody := `{"RequestId":"req-1","Code":"Throttling.User","Message":"slow down"}`
	tests := []struct {
		name string
		err  error
	}{
		{name: "runtime canonical constraint", err: &runtime.ConstraintViolationError{Parameter: "mode", Constraint: "enum"}},
		{name: "legacy canonical constraint", err: &ConstraintViolationError{Flag: "Mode", Constraint: "enum"}},
		{name: "runtime credential", err: &engine.CredentialError{Err: errors.New("profile not configured")}},
		{name: "host credential", err: &credentialConfigurationError{Err: errors.New("profile not configured")}},
		{name: "old SDK server", err: sdkerrors.NewServerError(429, serverBody, "")},
		{name: "Tea SDK server", err: tea.NewSDKError(map[string]interface{}{"statusCode": 503, "code": "ServiceUnavailable"})},
		{name: "network", err: &net.DNSError{Err: "temporary failure", Name: "ecs.aliyuncs.com", IsTemporary: true}},
		{name: "external plugin", err: &externalPluginError{err: errors.New("plugin process failed")}},
		{name: "postprocessing", err: errors.New("invalid --cli-query expression")},
		{name: "untyped usage wrapper", err: &engine.UsageError{Code: "INVALID_ARGUMENT", Err: errors.New("opaque parser failure")}},
		{name: "untyped internal", err: errors.New("AccessDenied Throttling InvalidAccessKeyId")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAgentErrorWithSearch(tt.err, []string{"ecs", "describe-instances"}, func(RecoverySearchRequest) bool { return true })
			assert.Same(t, tt.err, got)
			var agentErr *cli.AgentError
			assert.False(t, errors.As(got, &agentErr))
		})
	}
}

func TestExplicitLocalTypesAreEligibleForNonAIHint(t *testing.T) {
	tests := []error{
		&InvalidProductError{Code: "ecx"},
		&InvalidApiError{Name: "DescribeInstnaces", product: &meta.Product{Code: "ecs"}},
		&InvalidParameterError{Name: "InstnaceType", ProductCode: "ecs", ApiName: "RunInstances"},
		&argparser.UnknownFlagError{Flag: "instnace-type"},
		&runtime.MissingRequiredError{Flags: []string{"--region-id"}},
		&argparser.InvalidArgumentError{Err: errors.New("invalid argument")},
		&engine.InvalidOptionCombinationError{Err: errors.New("conflict")},
		&engine.InvalidHeaderError{Err: errors.New("bad header")},
		&engine.InvalidBodyFileError{Err: errors.New("unreadable")},
	}
	for _, err := range tests {
		assert.Truef(t, cli.IsAIRecoveryEligible(err), "%T", err)
	}
	assert.False(t, cli.IsAIRecoveryEligible(&runtime.ConstraintViolationError{Constraint: "enum"}))
	assert.False(t, cli.IsAIRecoveryEligible(&engine.CredentialError{Err: errors.New("credential")}))
}

func TestAgentErrorEnvelopeEndToEndIsOneCleanJSONDocument(t *testing.T) {
	testHome := t.TempDir()
	cleanupHome := setTestHomeDir(t, testHome)
	defer cleanupHome()
	writeMinimalConfigJSON(t, testHome)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"), []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, aimode.Save(filepath.Join(testHome, ".aliyun"), &aimode.AiConfig{Enabled: false}))
	t.Setenv(aimode.EnvAIMode, "")

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

	cli.DisableExitCode()
	defer cli.EnableExitCode()
	originalArgs := os.Args
	os.Args = []string{"aliyun", "ecs", "describe-instances", "--instnace-type", "ecs.g6.large", "--cli-ai-mode"}
	defer func() { os.Args = originalArgs }()
	cmd.Execute(ctx, os.Args[1:])

	assert.Empty(t, stdout.String())
	assert.Equal(t, 1, strings.Count(stderr.String(), "\n"))
	assert.NotContains(t, stderr.String(), "\x1b[")
	assert.NotContains(t, stderr.String(), cli.AIModeEnableTextHint)
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &decoded))
	assert.ElementsMatch(t, []string{"message", "did_you_mean", "recovery"}, mapKeys(decoded))
	assert.Equal(t, []interface{}{"--instance-type"}, decoded["did_you_mean"])
	recovery := decoded["recovery"].(map[string]interface{})
	assert.Equal(t, "search_parameter", recovery["action"])
	assert.Equal(t, "aliyun ecs describe-instances --help-search instance-type", recovery["command"])
}

func TestNonAIExplicitLocalErrorKeepsTextAndAppendsHintOnce(t *testing.T) {
	testHome := t.TempDir()
	cleanupHome := setTestHomeDir(t, testHome)
	defer cleanupHome()
	writeMinimalConfigJSON(t, testHome)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"), []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, aimode.Save(filepath.Join(testHome, ".aliyun"), &aimode.AiConfig{Enabled: true}))
	t.Setenv(aimode.EnvAIMode, "")
	t.Setenv("NO_COLOR", "1")

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
	assert.Equal(t, 1, strings.Count(stderr.String(), cli.AIModeEnableCommand))
	assert.True(t, strings.HasSuffix(stderr.String(), cli.AIModeEnableTextHint+"\n"))
}

func TestBuiltInSubcommandErrorUsesRootAIModeAdapter(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, " TRUE ")
	t.Setenv("NO_COLOR", "")
	cli.SetNoColorOverride(false)
	t.Cleanup(func() { cli.SetNoColorOverride(false) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	commando := NewCommando(stdout, config.Profile{Language: "en"})
	commando.InitWithCommand(root)

	configure := &cli.Command{
		Name: "configure",
		Run: func(ctx *cli.Context, args []string) error {
			return cli.NewInvalidCommandError(args[0], ctx)
		},
	}
	configure.AddSubCommand(&cli.Command{Name: "profile"})
	root.AddSubCommand(configure)

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	root.Execute(ctx, []string{"configure", "profiel"})

	assert.Empty(t, stdout.String())
	assert.NotContains(t, stderr.String(), "\x1b[")
	assert.NotContains(t, stderr.String(), cli.AIModeEnableTextHint)
	var envelope cli.AgentErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	assert.Equal(t, []string{"profile"}, envelope.DidYouMean)
	assert.Equal(t, "inspect_parent_help", envelope.Recovery.Action)
	assert.Equal(t, "aliyun configure --help", envelope.Recovery.Command)
	assert.Equal(t, "\x1b[0;31mtext\x1b[0m", cli.Colorized(cli.Red, "text"), "AI no-color override must be restored after Execute")
}

func TestBuiltInUnknownFlagFallsBackToCurrentCommandHelp(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "1")
	t.Setenv("NO_COLOR", "")
	cli.SetNoColorOverride(false)
	t.Cleanup(func() { cli.SetNoColorOverride(false) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	commando := NewCommando(stdout, config.Profile{Language: "en"})
	commando.InitWithCommand(root)
	root.AddSubCommand(&cli.Command{Name: "configure", Run: func(*cli.Context, []string) error { return nil }})

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	root.Execute(ctx, []string{"configure", "--bogus"})

	var envelope cli.AgentErrorEnvelope
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope))
	assert.Equal(t, "invalid flag --bogus", envelope.Message)
	assert.Equal(t, "inspect_command_help", envelope.Recovery.Action)
	assert.Equal(t, "aliyun configure --help", envelope.Recovery.Command)
	assert.NotContains(t, envelope.Recovery.Command, "--cli-section")
}

func TestBuiltInSubcommandErrorKeepsNonAIText(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "false")
	t.Setenv("NO_COLOR", "1")
	cli.SetNoColorOverride(false)
	t.Cleanup(func() { cli.SetNoColorOverride(false) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	commando := NewCommando(stdout, config.Profile{Language: "en"})
	commando.InitWithCommand(root)
	configure := &cli.Command{
		Name: "configure",
		Run: func(ctx *cli.Context, args []string) error {
			return cli.NewInvalidCommandError(args[0], ctx)
		},
	}
	root.AddSubCommand(configure)

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	root.Execute(ctx, []string{"configure", "profiel"})

	assert.Contains(t, stderr.String(), `ERROR: "profiel" is not a valid command`)
	assert.False(t, strings.HasPrefix(strings.TrimSpace(stderr.String()), "{"))
	assert.Equal(t, 1, strings.Count(stderr.String(), cli.AIModeEnableCommand))
}

func mapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestTypedLocalErrorsPreserveHumanText(t *testing.T) {
	cause := errors.New("original human message")
	tests := []error{
		&argparser.InvalidArgumentError{Flag: "--count", ExpectedType: "integer", Err: cause},
		&engine.InvalidOptionCombinationError{Options: []string{"--secure", "--insecure"}, Err: cause},
		&engine.InvalidHeaderError{Input: "broken", ExpectedFormat: "Name=Value", Err: cause},
		&engine.InvalidBodyFileError{Path: "/tmp/body.json", Err: cause},
	}
	for _, err := range tests {
		assert.Equal(t, cause.Error(), err.Error(), fmt.Sprintf("%T", err))
		assert.ErrorIs(t, err, cause)
	}
}
