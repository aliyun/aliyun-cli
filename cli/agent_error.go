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

type AgentErrorCategory string

const (
	InternalErrorCategory       AgentErrorCategory = "INTERNAL_ERROR"
	UsageErrorCategory          AgentErrorCategory = "USAGE_ERROR"
	AuthenticationErrorCategory AgentErrorCategory = "AUTHENTICATION_ERROR"
	PermissionErrorCategory     AgentErrorCategory = "PERMISSION_ERROR"
	ThrottlingErrorCategory     AgentErrorCategory = "THROTTLING_ERROR"
	NetworkErrorCategory        AgentErrorCategory = "NETWORK_ERROR"
	ServiceErrorCategory        AgentErrorCategory = "SERVICE_ERROR"
)

type AgentErrorRecovery struct {
	Command string `json:"command"`
}

type AgentErrorEnvelope struct {
	OK          bool               `json:"ok"`
	Category    AgentErrorCategory `json:"category"`
	Code        string             `json:"code"`
	Message     string             `json:"message"`
	Suggestions []string           `json:"suggestions"`
	Retryable   bool               `json:"retryable"`
	RequestID   string             `json:"requestId"`
	Recovery    AgentErrorRecovery `json:"recovery"`
}

type AgentError struct {
	envelope AgentErrorEnvelope
	cause    error
}

func NewAgentError(envelope AgentErrorEnvelope, cause error) *AgentError {
	envelope.Suggestions = copySuggestions(envelope.Suggestions)
	return &AgentError{envelope: envelope, cause: cause}
}

func (e *AgentError) Error() string {
	if e.envelope.Message != "" {
		return e.envelope.Message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.envelope.Code
}

func (e *AgentError) Unwrap() error {
	return e.cause
}

func (e *AgentError) Envelope() AgentErrorEnvelope {
	envelope := e.envelope
	envelope.Suggestions = copySuggestions(envelope.Suggestions)
	return envelope
}

func (e *AgentError) ExitCode() int {
	switch e.envelope.Category {
	case UsageErrorCategory:
		return 2
	case AuthenticationErrorCategory:
		return 3
	case PermissionErrorCategory:
		return 4
	case ThrottlingErrorCategory:
		return 5
	case NetworkErrorCategory:
		return 6
	case ServiceErrorCategory:
		return 7
	default:
		return 1
	}
}

func copySuggestions(suggestions []string) []string {
	copyOfSuggestions := make([]string, len(suggestions))
	copy(copyOfSuggestions, suggestions)
	return copyOfSuggestions
}
