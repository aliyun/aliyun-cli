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
	"strings"
)

const (
	AIModeEnableCommand  = "export ALIBABA_CLOUD_CLI_AI_MODE=1"
	AIModeEnableTextHint = "For AI agents, run:\n  " + AIModeEnableCommand +
		"\n\nThis enables compact Help, structured JSON errors, and actionable recovery guidance."
	aiModeEnableMessage = "Enable AI Mode for compact Help, structured JSON errors, and actionable recovery guidance."
)

// AIModeHint is embedded by Machine Help when the effective AI mode is off.
type AIModeHint struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

func NewAIModeHint() AIModeHint {
	return AIModeHint{
		Command: AIModeEnableCommand,
		Message: aiModeEnableMessage,
	}
}

type AgentErrorRecovery struct {
	Action  string `json:"action"`
	Command string `json:"command,omitempty"`
	Hint    string `json:"hint"`
}

type AgentErrorEnvelope struct {
	Message    string             `json:"message"`
	DidYouMean []string           `json:"did_you_mean,omitempty"`
	Recovery   AgentErrorRecovery `json:"recovery"`
}

type AgentError struct {
	envelope AgentErrorEnvelope
	cause    error
}

// NewAgentError returns nil when the required compact-envelope fields are
// incomplete. Optional data is normalized before it can reach JSON output.
func NewAgentError(envelope AgentErrorEnvelope, cause error) *AgentError {
	envelope.DidYouMean = compactStrings(envelope.DidYouMean)
	envelope.Recovery.Command = strings.TrimSpace(envelope.Recovery.Command)
	if strings.TrimSpace(envelope.Message) == "" ||
		strings.TrimSpace(envelope.Recovery.Action) == "" ||
		strings.TrimSpace(envelope.Recovery.Hint) == "" {
		return nil
	}
	return &AgentError{envelope: envelope, cause: cause}
}

func (e *AgentError) Error() string {
	if e.envelope.Message != "" {
		return e.envelope.Message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return ""
}

func (e *AgentError) Unwrap() error {
	return e.cause
}

func (e *AgentError) Envelope() AgentErrorEnvelope {
	envelope := e.envelope
	envelope.DidYouMean = copyStrings(envelope.DidYouMean)
	return envelope
}

// Supported agent-local errors are usage errors and retain the conventional
// command-line usage exit status.
func (e *AgentError) ExitCode() int { return 2 }

// AIRecoveryEligible is a structural marker implemented only by explicit CLI
// local error types. It intentionally has no text-based fallback.
type AIRecoveryEligible interface {
	AIRecoveryEligible()
}

func IsAIRecoveryEligible(err error) bool {
	if err == nil {
		return false
	}
	var eligible AIRecoveryEligible
	return errors.As(err, &eligible)
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copyOfValues := make([]string, len(values))
	copy(copyOfValues, values)
	return copyOfValues
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
