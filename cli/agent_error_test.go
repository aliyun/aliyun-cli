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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentErrorPreservesStableEnvelope(t *testing.T) {
	cause := errors.New("unknown flag --instnace-type")
	suggestions := []string{"--instance-type"}
	envelope := AgentErrorEnvelope{
		OK:          false,
		Category:    UsageErrorCategory,
		Code:        "UNKNOWN_FLAG",
		Message:     cause.Error(),
		Suggestions: suggestions,
		Retryable:   false,
		RequestID:   "",
		Recovery:    AgentErrorRecovery{Command: "aliyun ecs describe-instances --help"},
	}

	err := NewAgentError(envelope, cause)
	suggestions[0] = "--mutated"

	assert.Equal(t, cause.Error(), err.Error())
	assert.ErrorIs(t, err, cause)
	assert.Equal(t, []string{"--instance-type"}, err.Envelope().Suggestions)
	assert.Equal(t, 2, err.ExitCode())
}

func TestAgentErrorNormalizesNilSuggestions(t *testing.T) {
	err := NewAgentError(AgentErrorEnvelope{
		Category: InternalErrorCategory,
		Code:     "INTERNAL_ERROR",
		Message:  "opaque",
	}, errors.New("opaque"))

	require.NotNil(t, err.Envelope().Suggestions)
	assert.Empty(t, err.Envelope().Suggestions)
}

func TestAgentErrorExitCodes(t *testing.T) {
	tests := []struct {
		category AgentErrorCategory
		want     int
	}{
		{InternalErrorCategory, 1},
		{UsageErrorCategory, 2},
		{AuthenticationErrorCategory, 3},
		{PermissionErrorCategory, 4},
		{ThrottlingErrorCategory, 5},
		{NetworkErrorCategory, 6},
		{ServiceErrorCategory, 7},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			err := NewAgentError(AgentErrorEnvelope{Category: tt.category}, errors.New("test"))
			assert.Equal(t, tt.want, err.ExitCode())
		})
	}
}
