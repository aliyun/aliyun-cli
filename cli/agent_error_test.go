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

package cli

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentErrorPreservesCompactLocalEnvelope(t *testing.T) {
	cause := errors.New("unknown flag --instnace-type")
	suggestions := []string{"--instance-type"}
	envelope := AgentErrorEnvelope{
		Message:    cause.Error(),
		DidYouMean: suggestions,
		Recovery: AgentErrorRecovery{
			Action:  "search_parameter",
			Command: "aliyun ecs describe-instances --help-search instance-type",
			Hint:    "Search request parameters related to instance-type.",
		},
	}

	err := NewAgentError(envelope, cause)
	suggestions[0] = "--mutated"

	assert.Equal(t, cause.Error(), err.Error())
	assert.ErrorIs(t, err, cause)
	assert.Equal(t, []string{"--instance-type"}, err.Envelope().DidYouMean)
	assert.Equal(t, 2, err.ExitCode())

	encoded, marshalErr := json.Marshal(err.Envelope())
	require.NoError(t, marshalErr)
	assert.JSONEq(t, `{
		"message":"unknown flag --instnace-type",
		"did_you_mean":["--instance-type"],
		"recovery":{
			"action":"search_parameter",
			"command":"aliyun ecs describe-instances --help-search instance-type",
			"hint":"Search request parameters related to instance-type."
		}
	}`, string(encoded))
	for _, removed := range []string{"ok", "category", "code", "details", "suggestions", "requestId", "retryable"} {
		assert.NotContains(t, string(encoded), `"`+removed+`"`)
	}
}

func TestNewAgentErrorRejectsIncompleteRequiredEnvelope(t *testing.T) {
	valid := AgentErrorEnvelope{
		Message: "invalid local usage",
		Recovery: AgentErrorRecovery{
			Action: "inspect_request_help",
			Hint:   "Inspect the request help.",
		},
	}

	tests := []struct {
		name     string
		envelope AgentErrorEnvelope
	}{
		{name: "message", envelope: func() AgentErrorEnvelope {
			envelope := valid
			envelope.Message = " \t "
			return envelope
		}()},
		{name: "recovery action", envelope: func() AgentErrorEnvelope {
			envelope := valid
			envelope.Recovery.Action = ""
			return envelope
		}()},
		{name: "recovery hint", envelope: func() AgentErrorEnvelope {
			envelope := valid
			envelope.Recovery.Hint = "\n"
			return envelope
		}()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Nil(t, NewAgentError(test.envelope, errors.New("cause")))
		})
	}
}

func TestAgentErrorRecursivelyOmitsEmptyOptionalValues(t *testing.T) {
	err := NewAgentError(AgentErrorEnvelope{
		Message:    "invalid local usage",
		DidYouMean: []string{"", "  --instance-id  ", "\t"},
		Recovery: AgentErrorRecovery{
			Action:  "inspect_request_help",
			Command: " \t ",
			Hint:    "Inspect the request help.",
		},
	}, errors.New("invalid local usage"))
	require.NotNil(t, err)

	encoded, marshalErr := json.Marshal(err.Envelope())
	require.NoError(t, marshalErr)
	assert.JSONEq(t, `{
		"message":"invalid local usage",
		"did_you_mean":["--instance-id"],
		"recovery":{
			"action":"inspect_request_help",
			"hint":"Inspect the request help."
		}
	}`, string(encoded))
	assert.NotContains(t, string(encoded), `"command"`)
}

func TestAgentErrorOmitsEmptyOptionalFields(t *testing.T) {
	err := NewAgentError(AgentErrorEnvelope{
		Message: "missing required parameter(s): --region-id",
		Recovery: AgentErrorRecovery{
			Action: "inspect_request_help",
			Hint:   "Inspect the complete request help.",
		},
	}, errors.New("missing required parameter(s): --region-id"))

	encoded, marshalErr := json.Marshal(err.Envelope())
	require.NoError(t, marshalErr)
	assert.JSONEq(t, `{
		"message":"missing required parameter(s): --region-id",
		"recovery":{
			"action":"inspect_request_help",
			"hint":"Inspect the complete request help."
		}
	}`, string(encoded))
	assert.NotContains(t, string(encoded), "did_you_mean")
	assert.NotContains(t, string(encoded), "command")
}

func TestAIModeEnableHintsShareStableContent(t *testing.T) {
	assert.Equal(t, "export ALIBABA_CLOUD_CLI_AI_MODE=1", NewAIModeHint().Command)
	assert.Equal(t, "Enable AI Mode for compact Help, structured JSON errors, and actionable recovery guidance.", NewAIModeHint().Message)
	assert.Equal(t, "For AI agents, run:\n  export ALIBABA_CLOUD_CLI_AI_MODE=1\n\nThis enables compact Help, structured JSON errors, and actionable recovery guidance.", AIModeEnableTextHint)
}
