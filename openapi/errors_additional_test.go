package openapi

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalErrorContractsCoverMessagesMarkersAndUnwrap(t *testing.T) {
	cause := errors.New("original error")

	missing := &LegacyMissingRequiredError{Err: cause}
	assert.Equal(t, "original error", missing.Error())
	assert.ErrorIs(t, missing, cause)
	missing.AIRecoveryEligible()
	assert.Equal(t, "required parameters are not assigned", (*LegacyMissingRequiredError)(nil).Error())
	assert.NoError(t, (*LegacyMissingRequiredError)(nil).Unwrap())
	assert.NoError(t, newLegacyMissingRequiredError(nil))
	assert.Same(t, missing, newLegacyMissingRequiredError(missing))

	argument := &InvalidArgumentError{Err: cause}
	assert.Equal(t, "original error", argument.Error())
	assert.ErrorIs(t, argument, cause)
	argument.AIRecoveryEligible()
	assert.Equal(t, "invalid argument", (&InvalidArgumentError{}).Error())

	combination := &InvalidOptionCombinationError{Err: cause}
	assert.Equal(t, "original error", combination.Error())
	assert.ErrorIs(t, combination, cause)
	combination.AIRecoveryEligible()
	assert.Nil(t, combination.GetSuggestions())
	assert.Equal(t, "invalid option combination", (&InvalidOptionCombinationError{}).Error())

	header := &InvalidHeaderError{Err: cause}
	assert.Equal(t, "original error", header.Error())
	assert.ErrorIs(t, header, cause)
	header.AIRecoveryEligible()
	assert.Equal(t, "invalid header", (&InvalidHeaderError{}).Error())

	body := &InvalidBodyFileError{Err: cause}
	assert.Equal(t, "original error", body.Error())
	assert.ErrorIs(t, body, cause)
	body.AIRecoveryEligible()
	assert.Equal(t, "invalid body file", (&InvalidBodyFileError{}).Error())
}

func TestProductAPIAndParameterAgentContracts(t *testing.T) {
	product := &InvalidProductError{Code: "ECX"}
	assert.Equal(t, `"ecx" is not a valid command or product.`, product.AgentMessage())
	product.AIRecoveryEligible()
	assert.Nil(t, product.AgentSuggestions())

	api := &InvalidApiError{Name: "DescribeInstnaces", product: &meta.Product{
		Code: "ecs", ApiNames: []string{"DescribeInstances"},
	}}
	assert.Equal(t, `"DescribeInstnaces" is not a valid api.`, api.AgentMessage())
	api.AIRecoveryEligible()
	assert.Contains(t, api.AgentSuggestions(), "DescribeInstances")
	assert.Nil(t, (&InvalidApiError{Name: "missing"}).AgentSuggestions())

	flags := cli.NewFlagSet()
	flags.Add(&cli.Flag{Name: "region"})
	parameter := &InvalidParameterError{
		Name: "regoin", ProductCode: "ecs", ApiName: "DescribeInstances",
		ParameterNames: []string{"RegionId"}, flags: flags,
	}
	assert.Equal(t, `"--regoin" is not a valid parameter or flag.`, parameter.AgentMessage())
	parameter.AIRecoveryEligible()
	assert.NotEmpty(t, parameter.AgentSuggestions())

	pluginProduct := &InvalidProductOrPluginError{
		Code:    "ecx",
		plugins: []plugin.PluginInfo{{ProductCode: "ecs"}},
	}
	assert.Equal(t, `"ecx" is not a valid product.`, pluginProduct.AgentMessage())
	pluginProduct.AIRecoveryEligible()
	assert.Contains(t, pluginProduct.AgentSuggestions(), "ecs")

	unified := &InvalidUnifiedApiError{
		Name:    "describe-instnaces",
		product: &meta.Product{Code: "ecs", ApiNames: []string{"DescribeInstances"}},
		lPlugin: plugin.LocalPlugin{CmdNames: []string{"describe-instances"}},
	}
	assert.Equal(t, `"describe-instnaces" is not a valid api.`, unified.AgentMessage())
	unified.AIRecoveryEligible()
	assert.Contains(t, unified.AgentSuggestions(), "describe-instances")
	assert.Nil(t, (&InvalidUnifiedApiError{Name: "missing"}).AgentSuggestions())
}

func TestInvalidBaselineCommandContract(t *testing.T) {
	cause := errors.New("unknown command")
	err := &InvalidBaselineCommandError{
		Product: "ecs", Command: "describe-instnaces",
		Candidates: []string{"describe-instances"}, Err: cause,
	}
	assert.Contains(t, err.Error(), "unknown command")
	assert.ErrorIs(t, err, cause)
	err.AIRecoveryEligible()
	require.NotEmpty(t, err.GetSuggestions())
	require.NotEmpty(t, err.AgentSuggestions())

	fallback := &InvalidBaselineCommandError{Product: "ecs", Command: "bad"}
	assert.Contains(t, fallback.Error(), "invalid baseline command")
}
