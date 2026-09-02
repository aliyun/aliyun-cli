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
	"net/url"
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

func TestKebabProfileCaseMismatchSuggestsLowercaseProfile(t *testing.T) {
	args := []string{"ecs", "describe-instances", "--Profile", "default"}
	newCause := func() error {
		unknown := &argparser.UnknownFlagError{Flag: "Profile", Known: []string{"instance-type"}}
		return &engine.UsageError{
			Code: "UNKNOWN_FLAG",
			Err:  fmt.Errorf("%w (run `aliyun ecs describe-instances --help` for accepted flags)", unknown),
		}
	}

	newContext := func() (*Commando, *cli.Context) {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		cmd := &cli.Command{Name: "aliyun", EnableUnknownFlag: true}
		config.AddFlags(cmd.Flags())
		AddFlags(cmd.Flags())
		ctx.EnterCommand(cmd)
		commando := &Commando{
			profile:                 config.Profile{Language: "en"},
			recoverySearchValidator: func(RecoverySearchRequest) bool { return false },
		}
		return commando, ctx
	}

	t.Run("text", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "false")
		commando, ctx := newContext()
		got := commando.finishCommandRun(ctx, args, newCause())
		require.Error(t, got)
		assert.Contains(t, got.Error(), "unknown flag --Profile, did you mean --profile?")
	})

	t.Run("AI JSON", func(t *testing.T) {
		t.Setenv(aimode.EnvAIMode, "true")
		commando, ctx := newContext()
		got := commando.finishCommandRun(ctx, args, newCause())
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		encoded, err := json.Marshal(agentErr.Envelope())
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"message":"unknown flag --Profile",
			"did_you_mean":["--profile"],
			"recovery":{
				"action":"inspect_action_help",
				"command":"aliyun ecs describe-instances --help",
				"hint":"Inspect the action help and correct the parameter or flag."
			}
		}`, string(encoded))
	})
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
			return got.Keyword == "Instance"
		})

		assert.Equal(t, `"DescribeInstnaces" is not a valid api.`, envelope.Message)
		assert.Equal(t, []string{"DescribeInstances"}, envelope.DidYouMean)
		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs --help-search Instance", envelope.Recovery.Command)
		assert.Equal(t, "Search APIs related to Instance.", envelope.Recovery.Hint)
		assert.Equal(t, []RecoverySearchRequest{
			{Product: "ecs", Style: "pascal", Keyword: "Instance"},
		}, requests)
	})

	t.Run("short unknown API still uses validated product search", func(t *testing.T) {
		cause := &InvalidApiError{Name: "get", product: &meta.Product{Code: "sts"}}
		envelope := requireAgentEnvelope(t, cause, []string{"sts", "get"}, func(request RecoverySearchRequest) bool {
			assert.Equal(t, RecoverySearchRequest{Product: "sts", Style: "kebab", Keyword: "get"}, request)
			return true
		})

		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun sts --help-search get", envelope.Recovery.Command)
		assert.Equal(t, "Search APIs related to get.", envelope.Recovery.Hint)
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
		assert.Equal(t, "inspect_command_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun configure --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect commands under the current parent.", envelope.Recovery.Hint)
	})

	t.Run("unknown CLI subcommand keeps suggestions but never changes help into search", func(t *testing.T) {
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		parent := &cli.Command{Name: "configure"}
		parent.AddSubCommand(&cli.Command{Name: "profile"})
		ctx.EnterCommand(parent)
		cause := cli.NewInvalidCommandError("profiel", ctx)

		envelope := requireAgentEnvelope(t, cause, []string{"configure", "profiel"}, func(RecoverySearchRequest) bool {
			t.Fatal("UNKNOWN_COMMAND recovery must not validate a search command")
			return false
		})
		assert.Equal(t, []string{"profile"}, envelope.DidYouMean)
		assert.Equal(t, "inspect_command_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun configure --help", envelope.Recovery.Command)
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

	t.Run("PascalCase flag on a kebab command suggests the kebab flag", func(t *testing.T) {
		cause := &engine.UsageError{Code: "UNKNOWN_FLAG", Err: &argparser.UnknownFlagError{
			Flag: "InstanceType", Known: []string{"image-id", "instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, nil)

		assert.Equal(t, []string{"--instance-type"}, envelope.DidYouMean)
	})

	t.Run("kebab flag on a PascalCase command suggests the PascalCase flag", func(t *testing.T) {
		cause := &InvalidParameterError{
			Name:           "instance-type",
			ProductCode:    "ecs",
			ApiName:        "DescribeInstances",
			ParameterNames: []string{"ImageId", "InstanceType"},
		}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstances"}, nil)

		assert.Equal(t, []string{"--InstanceType"}, envelope.DidYouMean)
	})

	t.Run("missing required parameter uses complete request help", func(t *testing.T) {
		cause := &engine.UsageError{Code: "MISSING_REQUIRED_PARAMETER", Err: &runtime.MissingRequiredError{
			Flags: []string{"--image-id", "--instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, nil)

		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs run-instances --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the API help for request parameters and provide every required value.", envelope.Recovery.Hint)
	})

	t.Run("missing required parameter hides PascalCase wire paths", func(t *testing.T) {
		cause := &engine.UsageError{Code: "MISSING_REQUIRED_PARAMETER", Err: &runtime.MissingRequiredError{
			Flags: []string{"--user-name"},
			Paths: []string{"UserName"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ram", "create-user"}, nil)

		assert.Equal(t, "missing required parameter(s): --user-name", envelope.Message)
	})

	t.Run("legacy missing required parameter uses complete request help", func(t *testing.T) {
		cause := cli.NewErrorWithTip(&LegacyMissingRequiredError{
			Err: errors.New("required parameters not assigned: --InstanceId"),
		}, "use `aliyun ecs DescribeInstanceAttribute --help` to get more information")
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstanceAttribute"}, nil)

		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "required parameters not assigned: --InstanceId", envelope.Message)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeInstanceAttribute --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the API help for request parameters and provide every required value.", envelope.Recovery.Hint)
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
		assert.Equal(t, "aliyun ecs describe-instances --help", envelope.Recovery.Command)
		var agentErr *cli.AgentError
		require.ErrorAs(t, normalizeAgentError(cause, []string{"ecs", "describe-instances"}), &agentErr)
		assert.Equal(t, 2, agentErr.ExitCode())
		assert.Equal(t, "Remove one of the conflicting options: --cli-dry-run-json, --pager.", envelope.Recovery.Hint)
	})

	t.Run("host CheckFlags option conflict is normalized", func(t *testing.T) {
		command := &cli.Command{Name: "aliyun"}
		command.Flags().Add(&cli.Flag{Name: "cli-dry-run-json", AssignedMode: cli.AssignedNone, ExcludeWith: []string{"pager"}})
		command.Flags().Add(&cli.Flag{Name: "pager", AssignedMode: cli.AssignedNone})
		ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
		ctx.EnterCommand(command)
		ctx.Flags().Get("cli-dry-run-json").SetAssigned(true)
		ctx.Flags().Get("pager").SetAssigned(true)

		cause := ctx.CheckFlags()
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstances", "--cli-dry-run-json", "--pager"}, nil)
		assert.Equal(t, "flag --cli-dry-run-json is exclusive with --pager", envelope.Message)
		assert.Equal(t, "fix_option_combination", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeInstances --help", envelope.Recovery.Command)
	})

	t.Run("section all conflict recovers with same section search", func(t *testing.T) {
		text := "--cli-section does not support --help-all without --help-search"
		cause := &engine.UsageError{Code: "INVALID_OPTION_COMBINATION", Err: &engine.InvalidOptionCombinationError{
			Options: []string{"--cli-section", "--help-all"}, Err: errors.New(text),
		}}
		envelope := requireAgentEnvelope(t, cause, []string{
			"help", "ecs", "DescribeInstances", "--version", "2014-05-26",
			"--cli-section", "request", "--cli-output", "json",
		}, nil)

		assert.Equal(t, text, envelope.Message)
		assert.Equal(t, "fix_option_combination", envelope.Recovery.Action)
		assert.Equal(t,
			"aliyun help ecs DescribeInstances --version 2014-05-26 --cli-section request --help-search <keyword> --cli-output json",
			envelope.Recovery.Command)
		assert.Equal(t, "Search the same Help section with a keyword instead of using --help-all alone.", envelope.Recovery.Hint)
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
		assert.Equal(t, "--body-file: /private/secret/request.json: permission denied", envelope.Message)
		assert.Equal(t, "aliyun ecs run-instances --body-file --help", envelope.Recovery.Command)
		assert.NotContains(t, envelope.Recovery.Command, "/private/secret")
		assert.NotContains(t, envelope.Recovery.Hint, "/private/secret")
		assert.Equal(t, "Check that --body-file points to a readable file.", envelope.Recovery.Hint)
	})
}

func TestNormalizeAgentErrorRedactsHeaderValuesButKeepsBodyFilePathsOnlyInMessage(t *testing.T) {
	tests := []struct {
		name      string
		cause     error
		sensitive string
		message   string
	}{
		{
			name:      "runtime header",
			cause:     &engine.InvalidHeaderError{Err: errors.New(`invalid header "Authorization=secret-token"`)},
			sensitive: "secret-token",
			message:   "invalid --header value: expected Name=Value",
		},
		{
			name:      "legacy header",
			cause:     &InvalidHeaderError{Err: errors.New(`invalid header "Authorization=legacy-secret"`)},
			sensitive: "legacy-secret",
			message:   "invalid --header value: expected Name=Value",
		},
		{
			name:      "runtime body file",
			cause:     &engine.InvalidBodyFileError{Path: "/private/customer/request.json", Err: errors.New("--body-file: open /private/customer/request.json: permission denied")},
			sensitive: "/private/customer/request.json",
			message:   "--body-file: open /private/customer/request.json: permission denied",
		},
		{
			name:      "legacy body file",
			cause:     &InvalidBodyFileError{Path: "/tmp/customer-body.json", Err: errors.New("--body-file: open /tmp/customer-body.json: no such file")},
			sensitive: "/tmp/customer-body.json",
			message:   "--body-file: open /tmp/customer-body.json: no such file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := requireAgentEnvelope(t, tt.cause, []string{"ecs", "DescribeInstances"}, nil)
			assert.Equal(t, tt.message, envelope.Message)
			if strings.Contains(tt.name, "body file") {
				assert.Contains(t, envelope.Message, tt.sensitive)
				assert.NotContains(t, envelope.Recovery.Command, tt.sensitive)
				assert.NotContains(t, envelope.Recovery.Hint, tt.sensitive)
			} else {
				encoded, err := json.Marshal(envelope)
				require.NoError(t, err)
				assert.NotContains(t, string(encoded), tt.sensitive)
			}
		})
	}
}

func TestNormalizeAgentErrorSearchValidationFallbackAndCommandStyle(t *testing.T) {
	t.Run("baseline wrapper wins over its canonical unknown API cause", func(t *testing.T) {
		cause := &InvalidBaselineCommandError{
			Product:    "sts",
			Command:    "get-caller",
			Candidates: []string{"assume-role", "get-caller-identity"},
			Err:        &InvalidApiError{Name: "get-caller"},
		}
		envelope := requireAgentEnvelope(t, cause, []string{"sts", "get-caller"}, func(got RecoverySearchRequest) bool {
			return got.Style == "kebab" && got.Keyword == "caller-identity"
		})

		assert.Equal(t, []string{"get-caller-identity"}, envelope.DidYouMean)
		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun sts --help-search caller-identity", envelope.Recovery.Command)
	})

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
		assert.Equal(t, []RecoverySearchRequest{{Product: "bssopenapi", Style: "pascal", Keyword: "Bill"}}, requests)
	})

	for _, input := range []string{"DescribeCoupons", "QueryAvailableCoupons", "QueryCouponDetails"} {
		t.Run("coupon token recovery for "+input, func(t *testing.T) {
			cause := &InvalidApiError{
				Name: input,
				product: &meta.Product{Code: "bssopenapi", ApiNames: []string{
					"QueryAccountBalance", "QueryAccountDetails", "QueryCashCoupons", "QueryOrders",
				}},
			}
			envelope := requireAgentEnvelope(t, cause, []string{"bssopenapi", input}, func(got RecoverySearchRequest) bool {
				return got.Keyword == "Coupon"
			})

			assert.Equal(t, []string{"QueryCashCoupons"}, envelope.DidYouMean)
			assert.Equal(t, "search_api", envelope.Recovery.Action)
			assert.Equal(t, "aliyun bssopenapi --help-search Coupon", envelope.Recovery.Command)
			assert.Equal(t, "Search APIs related to Coupon.", envelope.Recovery.Hint)
		})
	}

	t.Run("kebab unknown API keeps kebab help in the recovery command", func(t *testing.T) {
		cause := &InvalidBaselineCommandError{
			Product: "ecs",
			Command: "describe-instnaces",
			Err:     errors.New(`"describe-instnaces" is not a valid api.`),
		}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instnaces"}, func(RecoverySearchRequest) bool { return false })

		assert.Equal(t, "inspect_product_help", envelope.Recovery.Action)
		assert.Equal(t, "ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help", envelope.Recovery.Command)
		assert.Equal(t, "Inspect the available APIs for this product.", envelope.Recovery.Hint)
	})

	t.Run("kebab API search recovery keeps the kebab help env prefix", func(t *testing.T) {
		cause := &InvalidBaselineCommandError{
			Product: "ecs",
			Command: "describe-instance",
			Err:     errors.New(`"describe-instance" is not a valid api.`),
		}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instance"}, func(got RecoverySearchRequest) bool {
			return got.Style == "kebab" && got.Keyword == "instance"
		})

		assert.Equal(t, "search_api", envelope.Recovery.Action)
		assert.Equal(t, "ALIBABA_CLOUD_BASELINE_PRODUCT_HELP=true aliyun ecs --help-search instance", envelope.Recovery.Command)
		assert.Equal(t, "Search APIs related to instance.", envelope.Recovery.Hint)
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

	t.Run("required recovery keeps API help and explicit API version", func(t *testing.T) {
		cause := &runtime.MissingRequiredError{Flags: []string{"--instance-id"}}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "describe-instance-attribute", "--api-version=2014-05-26"}, nil)

		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs describe-instance-attribute --api-version 2014-05-26 --help", envelope.Recovery.Command)
	})
}

func TestNormalizeAgentErrorConstraintViolations(t *testing.T) {
	t.Run("runtime enum violation exposes allowed values", func(t *testing.T) {
		cause := &engine.UsageError{
			Code: "INVALID_PARAMETER_VALUE",
			Err: &runtime.ConstraintViolationError{
				Parameter:  "Status",
				Flag:       "--status",
				Path:       "Status",
				Constraint: "enum",
				Actual:     "runing",
				Allowed:    []string{"Running", "Stopped"},
			},
		}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "describe-instances", "--status", "runing"}, nil)

		assert.Equal(t, `--status value "runing" is not allowed`, envelope.Message)
		assert.Equal(t, []string{"Running", "Stopped"}, envelope.DidYouMean)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun help ecs describe-instances --cli-section request", envelope.Recovery.Command)
		assert.Equal(t, "Use one of the allowed values for --status.", envelope.Recovery.Hint)
	})

	t.Run("runtime maximum violation keeps the bound in message and hint", func(t *testing.T) {
		cause := &runtime.ConstraintViolationError{
			Parameter:  "PageSize",
			Flag:       "--page-size",
			Constraint: "maximum",
			Actual:     "101",
			Expected:   "100",
		}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "describe-instances", "--page-size", "101"}, nil)

		assert.Equal(t, `--page-size value "101" must be less than or equal to 100`, envelope.Message)
		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "Use a value less than or equal to 100.", envelope.Recovery.Hint)
	})

	t.Run("legacy enum violation exposes allowed values", func(t *testing.T) {
		cause := &ConstraintViolationError{
			Flag:       "Status",
			Value:      "runing",
			Constraint: "enum",
			Allowed:    []string{"Running", "Stopped"},
		}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "DescribeInstances", "--Status", "runing"}, nil)

		assert.Equal(t, `--Status value "runing" is not allowed`, envelope.Message)
		assert.Equal(t, []string{"Running", "Stopped"}, envelope.DidYouMean)
		assert.Equal(t, "aliyun help ecs DescribeInstances --cli-section request", envelope.Recovery.Command)
	})

	t.Run("legacy maximum violation keeps the bound", func(t *testing.T) {
		cause := &ConstraintViolationError{
			Flag:       "PageSize",
			Value:      "101",
			Constraint: "maximum",
			Maximum:    "100",
		}
		envelope := requireAgentEnvelope(t, cause,
			[]string{"ecs", "DescribeInstances", "--PageSize", "101"}, nil)

		assert.Equal(t, `--PageSize value "101" must be less than or equal to 100`, envelope.Message)
		assert.Equal(t, "Use a value less than or equal to 100.", envelope.Recovery.Hint)
	})
}

func TestNormalizeAgentErrorStrictlyBypassesExcludedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "runtime credential", err: &engine.CredentialError{Err: errors.New("profile not configured")}},
		{name: "host credential", err: &credentialConfigurationError{Err: errors.New("profile not configured")}},
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

func TestNormalizeAgentErrorOAuthRefreshFailure(t *testing.T) {
	// The invoker wraps credential init failures as "init client failed: %w";
	// the OAuth refresh error must surface through that chain with the OAuth
	// server's facts and a re-login recovery.
	cause := fmt.Errorf("init client failed: %w", &config.OAuthTokenError{
		Stage:       "failed to refresh token",
		StatusCode:  400,
		Code:        "invalid_grant",
		Description: "invalid refreshToken",
		RequestID:   "52beb483-d0bb-483d-bcfa-049bab9e2f6d",
		ReLogin:     "aliyun configure --mode OAuth --oauth-site-type CN --profile oauth",
	})
	envelope := requireAgentEnvelope(t, cause, []string{"sts", "GetCallerIdentity"}, nil)

	assert.Equal(t, "failed to refresh token: invalid_grant (invalid refreshToken)", envelope.Message)
	assert.Equal(t, "invalid_grant", envelope.ErrorCode)
	assert.Equal(t, 400, envelope.StatusCode)
	assert.Equal(t, "52beb483-d0bb-483d-bcfa-049bab9e2f6d", envelope.RequestId)
	assert.Equal(t, "reauthenticate", envelope.Recovery.Action)
	assert.Equal(t, "aliyun configure --mode OAuth --oauth-site-type CN --profile oauth", envelope.Recovery.Command)
	assert.NotEmpty(t, envelope.Recovery.Hint)
}

func TestNormalizeAgentErrorServerErrorsShareOneEnvelope(t *testing.T) {
	serverBody := `{"RequestId":"req-1","Code":"Throttling.User","Message":"slow down"}`
	tests := []struct {
		name          string
		err           error
		wantMessage   string
		wantCode      string
		wantStatus    int
		wantRequestID string
	}{
		{
			name:          "old SDK server keeps request id",
			err:           sdkerrors.NewServerError(429, serverBody, ""),
			wantMessage:   "slow down",
			wantCode:      "Throttling.User",
			wantStatus:    429,
			wantRequestID: "req-1",
		},
		{
			name:        "Tea SDK server",
			err:         tea.NewSDKError(map[string]interface{}{"statusCode": 503, "code": "ServiceUnavailable", "message": "try later"}),
			wantMessage: "try later",
			wantCode:    "ServiceUnavailable",
			wantStatus:  503,
		},
		{
			name:        "Tea SDK server wrapped by runtime call failure",
			err:         fmt.Errorf("runtime: call failed: %w", tea.NewSDKError(map[string]interface{}{"statusCode": 503, "code": "ServiceUnavailable", "message": "try later"})),
			wantMessage: "try later",
			wantCode:    "ServiceUnavailable",
			wantStatus:  503,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAgentErrorWithSearch(tt.err, []string{"ecs", "describe-instances"}, nil)
			var agentErr *cli.AgentError
			require.ErrorAs(t, got, &agentErr)
			envelope := agentErr.Envelope()
			assert.Equal(t, tt.wantMessage, envelope.Message)
			assert.Equal(t, tt.wantCode, envelope.ErrorCode)
			assert.Equal(t, tt.wantStatus, envelope.StatusCode)
			assert.Equal(t, tt.wantRequestID, envelope.RequestId)
			assert.Equal(t, "diagnose_error_code", envelope.Recovery.Action)
			assert.Contains(t, envelope.Recovery.Command, "openapiexplorer get-error-code-solutions")
			assert.Contains(t, envelope.Recovery.Command, "--error-code '"+tt.wantCode+"'")
			assert.Contains(t, envelope.Recovery.Command, "--product Ecs")
			assert.Equal(t, 2, agentErr.ExitCode())
		})
	}
}

func TestServerAgentErrorDiagnoseCommandMatchesCallerStyle(t *testing.T) {
	makeErr := func() error {
		return tea.NewSDKError(map[string]interface{}{"statusCode": 400, "code": "InvalidParameter", "message": "bad"})
	}

	t.Run("camel caller gets PascalCase entry with raw-name flags", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(makeErr(), []string{"ecs", "DescribeInstances"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		assert.Equal(t,
			"aliyun openapiexplorer GetErrorCodeSolutions --errorCode 'InvalidParameter' --product Ecs --acceptLanguage en-US --region cn-hangzhou",
			agentErr.Envelope().Recovery.Command)
	})

	t.Run("kebab caller keeps kebab entry", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(makeErr(), []string{"ecs", "describe-instances"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		assert.Equal(t,
			"aliyun openapiexplorer get-error-code-solutions --error-code 'InvalidParameter' --product Ecs --accept-language en-US --region cn-hangzhou",
			agentErr.Envelope().Recovery.Command)
	})
}

func TestServerErrorCodeParameter(t *testing.T) {
	tests := map[string]string{
		"InvalidParameter.Tags":                 "Tags",
		"MissingParameter.Tags":                 "Tags",
		"MissingParameter.ResourceARN":          "ResourceARN",
		"InvalidParameter.ResourceARN.1":        "",
		"InvalidParameter":                      "",
		"InvalidParameter.":                     "",
		"InvalidAccessKeyId.NotFound":           "",
		"InvalidOperation.NotSupportedEndpoint": "",
		"Throttling.User":                       "",
	}
	for code, want := range tests {
		t.Run(code, func(t *testing.T) {
			assert.Equal(t, want, serverErrorCodeParameter(code))
		})
	}
}

func TestServerAgentErrorParameterCodeAddsParameterGuidance(t *testing.T) {
	makeErr := func(code string) error {
		return tea.NewSDKError(map[string]interface{}{"statusCode": 400, "code": code, "message": "bad parameter"})
	}

	t.Run("camel caller gets parameter-scoped help flag", func(t *testing.T) {
		envelope := requireAgentEnvelope(t, makeErr("InvalidParameter.Tags"), []string{"tag", "TagResources"}, nil)
		assert.Equal(t, "diagnose_error_code", envelope.Recovery.Action)
		assert.Contains(t, envelope.Recovery.Command, "openapiexplorer GetErrorCodeSolutions")
		assert.Equal(t,
			"Look up diagnostic solutions for error code InvalidParameter.Tags. For the accepted format of parameter Tags, run: aliyun tag TagResources --Tags --help",
			envelope.Recovery.Hint)
	})

	t.Run("kebab caller gets parameter keyword search", func(t *testing.T) {
		envelope := requireAgentEnvelope(t, makeErr("MissingParameter.Tags"), []string{"tag", "tag-resources"}, nil)
		assert.Equal(t, "diagnose_error_code", envelope.Recovery.Action)
		assert.Equal(t,
			"Look up diagnostic solutions for error code MissingParameter.Tags. For the accepted format of parameter Tags, run: aliyun tag tag-resources --help-search Tags",
			envelope.Recovery.Hint)
	})

	t.Run("non-parameter code keeps the plain diagnose hint", func(t *testing.T) {
		envelope := requireAgentEnvelope(t, makeErr("InvalidAccessKeyId.NotFound"), []string{"ecs", "DescribeInstances"}, nil)
		assert.Equal(t,
			"Look up diagnostic solutions for error code InvalidAccessKeyId.NotFound.",
			envelope.Recovery.Hint)
	})

	t.Run("without an action the plain diagnose hint stays", func(t *testing.T) {
		envelope := requireAgentEnvelope(t, makeErr("InvalidParameter.Tags"), []string{"ecs"}, nil)
		assert.Equal(t,
			"Look up diagnostic solutions for error code InvalidParameter.Tags.",
			envelope.Recovery.Hint)
	})
}

func TestServerAgentErrorCleansTeaMessageEnvelope(t *testing.T) {
	// darabonba-openapi builds Tea messages as "code: <status>, <msg> request id: <id>";
	// the duplicates must be stripped and the request id surfaced as a field.
	err := tea.NewSDKError(map[string]interface{}{
		"statusCode": 400,
		"code":       "InvalidOperation.NotSupportedEndpoint",
		"message":    "code: 400, The specified endpoint can't operate this region. request id: 01A04DF0-7C9A-55A7",
	})
	got := normalizeAgentErrorWithSearch(err, []string{"ecs", "describe-instances"}, nil)
	var agentErr *cli.AgentError
	require.ErrorAs(t, got, &agentErr)
	envelope := agentErr.Envelope()
	assert.Equal(t, "The specified endpoint can't operate this region.", envelope.Message)
	assert.Equal(t, 400, envelope.StatusCode)
	assert.Equal(t, "01A04DF0-7C9A-55A7", envelope.RequestId)
}

func TestServerAgentErrorFallsBackToActionHelpWithoutCode(t *testing.T) {
	// A server error with no code keeps the generic action-help recovery.
	err := tea.NewSDKError(map[string]interface{}{"statusCode": 500, "message": "internal"})
	got := normalizeAgentErrorWithSearch(err, []string{"ecs", "describe-instances"}, nil)
	var agentErr *cli.AgentError
	require.ErrorAs(t, got, &agentErr)
	envelope := agentErr.Envelope()
	assert.Equal(t, "internal", envelope.Message)
	assert.Equal(t, "inspect_action_help", envelope.Recovery.Action)
	assert.Equal(t, "aliyun ecs describe-instances --help", envelope.Recovery.Command)
}

func TestNormalizeAgentErrorServerErrorKeepsTipPath(t *testing.T) {
	tipped := cli.NewErrorWithTip(
		sdkerrors.NewServerError(400, `{"Code":"PricingNotSupported","Message":"no pricing"}`, ""),
		"this OpenAPI either incurs no cost or has no pricing mapping registered yet",
	)
	got := normalizeAgentErrorWithSearch(tipped, []string{"ecs", "describe-instances"}, nil)
	assert.Same(t, tipped, got)
}

func TestNormalizeAgentErrorEndpointResolution(t *testing.T) {
	// Client-side endpoint resolution failures are wrapped exactly as invoker.go
	// wraps them (ErrorWithTip around a %w chain) and must become an envelope
	// whose diagnostics command matches the caller's command style.
	newEndpointError := func() error {
		endpointErr := &meta.InvalidEndpointError{Region: "invalid-region-xxx", Product: &meta.Product{}}
		return cli.NewErrorWithTip(
			fmt.Errorf("unknown endpoint for %s/%s! failed %w", "ecs", "invalid-region-xxx", endpointErr),
			"Use flag --endpoint xxx.aliyuncs.com to assign endpoint",
		)
	}

	t.Run("kebab command gets kebab diagnostics", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(newEndpointError(), []string{"ecs", "describe-zones"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		envelope := agentErr.Envelope()
		assert.Contains(t, envelope.Message, "unknown endpoint for region invalid-region-xxx")
		assert.Equal(t, "fix_endpoint_or_region", envelope.Recovery.Action)
		assert.Equal(t, "aliyun openapiexplorer get-product-endpoints --product Ecs --region cn-hangzhou "+endpointProjection, envelope.Recovery.Command)
		assert.NotEmpty(t, envelope.Recovery.Hint)
		assert.Equal(t, 2, agentErr.ExitCode())
	})

	t.Run("camel command gets camel diagnostics", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(newEndpointError(), []string{"ecs", "DescribeZones"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		envelope := agentErr.Envelope()
		assert.Equal(t, "aliyun openapiexplorer GetProductEndpoints --product Ecs --region cn-hangzhou "+endpointProjection, envelope.Recovery.Command)
	})
}

const endpointProjection = "--cli-query 'data.endpoints[*].{regionId:regionId,endpoint:endpoint}'"

func TestSanitizeNetworkTransportErrorStripsSignedURL(t *testing.T) {
	signedURL := "https://ecs.cn-shanghai.aliyuncs.com/?AccessKeyId=REAL-AK&Signature=abc%2Fsig&SignatureNonce=n1&RegionId=cn-shanghai"

	t.Run("bare url.Error", func(t *testing.T) {
		raw := &url.Error{Op: "Post", URL: signedURL, Err: errors.New("connection refused")}
		sanitized := sanitizeNetworkTransportError(raw)
		msg := sanitized.Error()
		assert.Contains(t, msg, "request to ecs.cn-shanghai.aliyuncs.com failed")
		assert.Contains(t, msg, "connection refused")
		assert.NotContains(t, msg, "AccessKeyId")
		assert.NotContains(t, msg, "REAL-AK")
		assert.NotContains(t, msg, "Signature")
		assert.NotContains(t, msg, "SignatureNonce")
	})

	t.Run("url.Error wrapped by retry-exhausted ClientError", func(t *testing.T) {
		inner := &url.Error{Op: "Post", URL: signedURL, Err: errors.New("connection refused")}
		clientErr := sdkerrors.NewClientError("SDK.TimeoutError", "timeout", inner)
		sanitized := sanitizeNetworkTransportError(clientErr)
		msg := sanitized.Error()
		assert.Contains(t, msg, "connection refused")
		assert.NotContains(t, msg, "AccessKeyId")
		assert.NotContains(t, msg, "REAL-AK")
	})

	t.Run("non-transport errors pass through", func(t *testing.T) {
		other := errors.New("some other failure")
		assert.Same(t, other, sanitizeNetworkTransportError(other))
	})
}

func TestNormalizeAgentErrorTransport(t *testing.T) {
	raw := &url.Error{Op: "Post", URL: "https://ecs.cn-shanghai.aliyuncs.com/?AccessKeyId=REAL-AK&Signature=sig", Err: errors.New("connection refused")}

	t.Run("camel", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(sanitizeNetworkTransportError(raw), []string{"ecs", "DescribeInstances"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		envelope := agentErr.Envelope()
		assert.Contains(t, envelope.Message, "request to ecs.cn-shanghai.aliyuncs.com failed")
		assert.NotContains(t, envelope.Message, "AccessKeyId")
		assert.Equal(t, "retry_or_fix_endpoint", envelope.Recovery.Action)
		assert.Contains(t, envelope.Recovery.Command, "GetProductEndpoints --product Ecs")
		assert.Equal(t, 2, agentErr.ExitCode())
	})

	t.Run("kebab", func(t *testing.T) {
		wrapped := fmt.Errorf("runtime: call failed: %w", raw)
		got := normalizeAgentErrorWithSearch(sanitizeNetworkTransportError(wrapped), []string{"ecs", "describe-instances"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		assert.Contains(t, agentErr.Envelope().Recovery.Command, "get-product-endpoints --product Ecs")
	})
}

func TestNormalizeAgentErrorQueryFilter(t *testing.T) {
	queryErr := &engine.QueryFilterError{Expr: "invalid[", Err: errors.New("SyntaxError: Incomplete expression")}

	t.Run("camel", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(queryErr, []string{"ecs", "DescribeZones"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		envelope := agentErr.Envelope()
		assert.Equal(t, `invalid --cli-query "invalid[": SyntaxError: Incomplete expression`, envelope.Message)
		assert.Equal(t, "fix_cli_query", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeZones --cli-section response", envelope.Recovery.Command)
		assert.Equal(t, 2, agentErr.ExitCode())
	})

	t.Run("kebab", func(t *testing.T) {
		got := normalizeAgentErrorWithSearch(queryErr, []string{"ecs", "describe-zones"}, nil)
		var agentErr *cli.AgentError
		require.ErrorAs(t, got, &agentErr)
		assert.Equal(t, `invalid --cli-query "invalid[": SyntaxError: Incomplete expression`, agentErr.Envelope().Message)
		assert.Equal(t, "aliyun ecs describe-zones --cli-section response", agentErr.Envelope().Recovery.Command)
	})
}

func TestNormalizeAgentErrorKebabEndpointResolution(t *testing.T) {
	endpointErr := &runtime.EndpointNotResolvedError{Product: "ecs", Region: "invalid-region-for-ai-check"}
	got := normalizeAgentErrorWithSearch(endpointErr, []string{"ecs", "describe-instances"}, nil)
	var agentErr *cli.AgentError
	require.ErrorAs(t, got, &agentErr)
	envelope := agentErr.Envelope()
	assert.Contains(t, envelope.Message, `endpoint not resolved for product "ecs" region "invalid-region-for-ai-check"`)
	assert.Equal(t, "fix_endpoint_or_region", envelope.Recovery.Action)
	assert.Equal(t, "aliyun openapiexplorer get-product-endpoints --product Ecs --region cn-hangzhou "+endpointProjection, envelope.Recovery.Command)
	assert.Equal(t, 2, agentErr.ExitCode())
}

func TestNormalizeAgentHelpOptionErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *cli.HelpOptionError
		action string
		hint   string
	}{
		{
			name:   "conflict",
			err:    &cli.HelpOptionError{Code: cli.HelpOptionConflict, Option: "--help-all", ConflictsWith: "--help"},
			action: "fix_option_combination",
			hint:   "Use only one Help operation; remove either --help-all or --help.",
		},
		{
			name:   "empty search",
			err:    &cli.HelpOptionError{Code: cli.HelpOptionEmptySearch, Option: "--help-search"},
			action: "fix_help_options",
			hint:   "Provide a non-empty query after --help-search.",
		},
		{
			name:   "invalid output",
			err:    &cli.HelpOptionError{Code: cli.HelpOptionInvalidOutput, Option: "--cli-output", Value: "yaml"},
			action: "fix_help_options",
			hint:   "Use --cli-output json, or remove --cli-output.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := requireAgentEnvelope(t, tt.err, []string{"ecs", "DescribeInstances"}, nil)
			assert.Equal(t, tt.action, envelope.Recovery.Action)
			assert.Equal(t, tt.hint, envelope.Recovery.Hint)
			assert.Empty(t, envelope.Recovery.Command)
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
		&cli.HelpOptionError{Code: cli.HelpOptionConflict},
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

func TestCLIOutputJSONStructuresLocalErrorWhenAIModeIsDisabled(t *testing.T) {
	testHome := t.TempDir()
	cleanupHome := setTestHomeDir(t, testHome)
	defer cleanupHome()
	writeMinimalConfigJSON(t, testHome)
	require.NoError(t, os.MkdirAll(filepath.Join(testHome, ".aliyun", "plugins"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(testHome, ".aliyun", "plugins", "manifest.json"), []byte(`{"plugins":{}}`), 0644))
	require.NoError(t, aimode.Save(filepath.Join(testHome, ".aliyun"), &aimode.AiConfig{Enabled: false}))
	t.Setenv(aimode.EnvAIMode, "")
	t.Setenv("NO_COLOR", "")

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
	os.Args = []string{"aliyun", "ecs", "describe-instances", "--instnace-type", "ecs.g6.large", "--cli-output", "json", "--no-cli-ai-mode"}
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
	assert.Equal(t, "inspect_command_help", envelope.Recovery.Action)
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

func TestSectionHelpAllConflictPrintsTextRecoveryAndAIModeHint(t *testing.T) {
	t.Setenv(aimode.EnvAIMode, "false")
	t.Setenv("NO_COLOR", "1")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root := &cli.Command{
		Name: "aliyun", EnableUnknownFlag: true,
		Run: func(*cli.Context, []string) error {
			return &InvalidOptionCombinationError{
				Options: []string{"--cli-section", "--help-all"},
				Err:     errors.New("Help sections do not support --help-all"),
			}
		},
	}
	config.AddFlags(root.Flags())
	AddFlags(root.Flags())
	CliHelpSectionFlag(root.Flags()).SetAssigned(true)
	CliHelpSectionFlag(root.Flags()).SetValue("request")
	CliHelpAllFlag(root.Flags()).SetAssigned(true)
	commando := NewCommando(stdout, config.Profile{Language: "en"})
	commando.InitWithCommand(root)

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(root)
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	root.Execute(ctx, []string{"help", "demo", "CreateReport"})

	assert.Contains(t, stderr.String(), "ERROR: Help sections do not support --help-all")
	assert.Contains(t, stderr.String(),
		"Search this Help:\n  aliyun help demo CreateReport --cli-section request --help-search <keyword> [--cli-output json]")
	assert.True(t, strings.HasSuffix(stderr.String(), cli.AIModeEnableTextHint+"\n"))
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

func TestNormalizeAgentErrorMissingRequiredAlwaysUsesAPIHelp(t *testing.T) {
	t.Run("runtime missing required ignores parameter search", func(t *testing.T) {
		cause := &engine.UsageError{Code: "MISSING_REQUIRED_PARAMETER", Err: &runtime.MissingRequiredError{
			Flags: []string{"--image-id", "--instance-type"},
		}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "run-instances"}, func(RecoverySearchRequest) bool {
			t.Fatal("missing required recovery must not validate a single-parameter search")
			return false
		})

		assert.Empty(t, envelope.DidYouMean)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs run-instances --help", envelope.Recovery.Command)
	})

	t.Run("legacy doc required uses API help", func(t *testing.T) {
		cause := &LegacyDocRequiredError{Flags: []string{"--RegionId"}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstances"}, nil)

		assert.Equal(t, "missing required parameter(s): --RegionId", envelope.Message)
		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeInstances --help", envelope.Recovery.Command)
	})

	t.Run("validator rejection falls back to complete request help", func(t *testing.T) {
		cause := &LegacyDocRequiredError{Flags: []string{"--RegionId"}}
		envelope := requireAgentEnvelope(t, cause, []string{"ecs", "DescribeInstances"}, func(RecoverySearchRequest) bool {
			return false
		})

		assert.Equal(t, "inspect_request_help", envelope.Recovery.Action)
		assert.Equal(t, "aliyun ecs DescribeInstances --help", envelope.Recovery.Command)
	})
}

func TestNormalizeAgentErrorConstraintUpgradesToTargetedSearch(t *testing.T) {
	cause := &engine.UsageError{Code: "INVALID_PARAMETER_VALUE", Err: &runtime.ConstraintViolationError{
		Flag: "--page-size", Actual: "101", Constraint: "maximum", Expected: "100",
	}}
	envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, func(request RecoverySearchRequest) bool {
		return request.Keyword == "page-size"
	})

	assert.Equal(t, "search_parameter", envelope.Recovery.Action)
	assert.Equal(t, "aliyun ecs describe-instances --help-search page-size", envelope.Recovery.Command)
}

func TestNormalizeAgentErrorExternalFlagReject(t *testing.T) {
	cause := &engine.UsageError{Code: "INVALID_ARGUMENT", Err: &argparser.ExternalFlagRejectError{
		Flag:    "RegionId",
		Message: "--RegionId is only supported by legacy PascalCase commands; use --region instead",
	}}
	envelope := requireAgentEnvelope(t, cause, []string{"ecs", "describe-instances"}, nil)

	assert.Equal(t, "--RegionId is only supported by legacy PascalCase commands; use --region instead", envelope.Message)
	assert.Equal(t, "inspect_action_help", envelope.Recovery.Action)
	assert.Equal(t, "aliyun ecs describe-instances --help", envelope.Recovery.Command)
}
