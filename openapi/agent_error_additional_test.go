package openapi

import (
	"errors"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAgentErrorWrapperAndMessageFallbacks(t *testing.T) {
	assert.Nil(t, normalizeAgentError(nil, nil))
	plain := errors.New("plain failure")
	assert.Same(t, plain, normalizeAgentError(plain, []string{"ecs", "DescribeInstances"}))
	assert.Equal(t, " explicit ", nonEmptyMessage(" explicit ", plain))
	assert.Equal(t, "plain failure", nonEmptyMessage("", plain))
	assert.Equal(t, "CLI request is invalid.", nonEmptyMessage("", nil))
	assert.Equal(t, "first", firstString([]string{"first", "second"}))
	assert.Empty(t, firstString(nil))
	assert.Equal(t, "value", firstNonEmpty("", " value ", "later"))
	assert.Empty(t, firstNonEmpty("", "  "))
}

func TestTeaRequestIDPayloadVariants(t *testing.T) {
	assert.Empty(t, teaRequestID(&tea.SDKError{}))
	assert.Empty(t, teaRequestID(&tea.SDKError{Data: tea.String("not-json")}))
	assert.Equal(t, "upper", teaRequestID(&tea.SDKError{Data: tea.String(`{"RequestId":"upper"}`)}))
	assert.Equal(t, "lower", teaRequestID(&tea.SDKError{Data: tea.String(`{"requestId":"lower"}`)}))
	assert.Empty(t, teaRequestID(&tea.SDKError{Data: tea.String(`{"requestId":42}`)}))
}

func TestOAuthAgentMessageVariants(t *testing.T) {
	assert.Equal(t, "refresh: invalid_grant (expired)", oauthAgentMessage(&config.OAuthTokenError{Stage: "refresh", Code: "invalid_grant", Description: "expired"}))
	assert.Equal(t, "refresh: invalid_grant", oauthAgentMessage(&config.OAuthTokenError{Stage: "refresh", Code: "invalid_grant"}))
	assert.Equal(t, "refresh (HTTP 400)", oauthAgentMessage(&config.OAuthTokenError{Stage: "refresh", StatusCode: 400}))
}

func TestConstraintMessagesHintsAndLegacyFactsAllKinds(t *testing.T) {
	tests := []struct {
		constraint string
		bound      string
		message    string
		hint       string
	}{
		{"enum", "", "not allowed", "allowed values"},
		{"minimum", "1", "greater than or equal", "greater than or equal"},
		{"maximum", "9", "less than or equal", "less than or equal"},
		{"minLength", "2", "at least", "at least"},
		{"maxLength", "8", "at most", "at most"},
		{"pattern", "^[a-z]+$", "does not match", "documented pattern"},
		{"future", "", "violates", "documented constraint"},
	}
	for _, test := range tests {
		facts := constraintFacts{flag: "Name", value: "bad", constraint: test.constraint, bound: test.bound}
		assert.Contains(t, constraintViolationMessage(facts), test.message, test.constraint)
		assert.Contains(t, constraintViolationHint(facts), test.hint, test.constraint)

		legacy := &ConstraintViolationError{Flag: "Name", Value: "bad", Constraint: test.constraint, Allowed: []string{"a"}}
		switch test.constraint {
		case "minimum":
			legacy.Minimum = test.bound
		case "maximum":
			legacy.Maximum = test.bound
		case "minLength":
			legacy.MinLength = test.bound
		case "maxLength":
			legacy.MaxLength = test.bound
		case "pattern":
			legacy.Pattern = test.bound
		}
		converted := legacyConstraintFacts(legacy)
		assert.Equal(t, test.bound, converted.bound, test.constraint)
		assert.Equal(t, []string{"a"}, converted.allowed)
	}

	for _, kind := range []string{"minimum", "maximum", "minLength", "maxLength"} {
		assert.Contains(t, constraintViolationHint(constraintFacts{flag: "Name", constraint: kind}), "documented constraint")
	}
}

func TestHelpOptionAgentErrorEveryRecovery(t *testing.T) {
	cause := errors.New("invalid options")
	assert.Same(t, cause, helpOptionAgentError(cause, nil))
	tests := []struct {
		code   cli.HelpOptionErrorCode
		action string
		hint   string
	}{
		{cli.HelpOptionConflict, "fix_option_combination", "Use only one"},
		{cli.HelpOptionDuplicate, "fix_option_combination", "duplicate"},
		{cli.HelpOptionEmptySearch, "fix_help_options", "non-empty"},
		{cli.HelpOptionInvalidOutput, "fix_help_options", "--cli-output json"},
		{cli.HelpOptionInvalidSection, "fix_help_options", "request or --cli-section response"},
		{cli.HelpOptionErrorCode("future"), "fix_help_options", "Correct the Help options"},
	}
	for _, test := range tests {
		option := &cli.HelpOptionError{Code: test.code, Option: "--help", ConflictsWith: "--help-all", Value: "bad"}
		err := helpOptionAgentError(cause, option)
		var agent *cli.AgentError
		require.ErrorAs(t, err, &agent)
		envelope := agent.Envelope()
		assert.Equal(t, test.action, envelope.Recovery.Action)
		assert.Contains(t, envelope.Recovery.Hint, test.hint)
	}
}

func TestRecoveryContextFallbackAndSuffixCommands(t *testing.T) {
	empty := newRecoveryContext(nil)
	assert.Equal(t, "aliyun --help", empty.productHelpCommand())
	assert.Equal(t, "aliyun --help", empty.productSearchCommand("bad token"))
	assert.Equal(t, "aliyun --help", empty.actionHelpCommand())
	assert.Equal(t, "aliyun --help", empty.parameterHelpCommand("Name"))
	assert.Empty(t, empty.parameterGuidanceCommand("Name"))
	assert.Equal(t, "aliyun --help", empty.requestHelpCommand())
	assert.Equal(t, "aliyun --help", empty.responseSectionCommand())

	context := newRecoveryContext([]string{"ecs", "DescribeInstances", "--version=2014-05-26"})
	assert.Equal(t, "aliyun ecs DescribeInstances --version 2014-05-26 --help", context.parameterHelpCommand("bad.name"))
	assert.Equal(t, context.requestHelpCommand(), context.requestSearchCommand("bad token"))
	assert.Equal(t, context.actionHelpCommand(), context.actionSearchCommand("bad token"))
	parent := newRecoveryContext([]string{"ecs", "DescribeInstances"})
	assert.Equal(t, "aliyun ecs --help", parent.parentHelpCommand("DescribeInstances"))

	assert.Equal(t, "aliyun --help", suffixHelpCommand("not-aliyun command"))
	assert.Equal(t, "aliyun --help-search profile", suffixSearchCommand("not-aliyun command", "profile"))
	assert.Equal(t, "aliyun --help", suffixSearchCommand("aliyun", "bad token"))
	assert.Nil(t, helpCommandParts("other ecs --help"))
	assert.Nil(t, helpCommandParts("aliyun ecs;rm --help"))
	assert.Equal(t, []string{"ecs", "DescribeInstances"}, helpCommandParts("aliyun help ecs DescribeInstances --help"))
}

func TestAPICandidateFormsForStyle(t *testing.T) {
	assert.Equal(t, []string{"describe-instances"}, apiCandidateFormsForStyle([]string{"DescribeInstances", "", "DescribeInstances"}, "kebab"))
	assert.Equal(t, []string{"DescribeInstances", "ListTags"}, apiCandidateFormsForStyle([]string{"describe-instances", "list_tags"}, "pascal"))
	assert.Equal(t, []string{"Already"}, apiCandidateFormsForStyle([]string{"Already"}, "unknown"))
}
