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

package telemetry

import (
	"errors"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/util"
)

const (
	schemaVersion                      = "1"
	EventTypeCommandCompleted          = "cli.command.completed"
	EventTypeConfigurationCompleted    = "cli.configuration.completed"
	EventTypePluginManagementCompleted = "cli.plugin.management.completed"
)

const (
	LocalErrorInvalidCommand           = "INVALID_COMMAND"
	LocalErrorInvalidFlag              = "INVALID_FLAG"
	LocalErrorMissingRequiredParameter = "MISSING_REQUIRED_PARAMETER"
	LocalErrorInvalidArgument          = "INVALID_ARGUMENT"
	LocalErrorInvalidOptionCombination = "INVALID_OPTION_COMBINATION"
	LocalErrorHelpOption               = "HELP_OPTION_ERROR"
	LocalErrorInvalidHeader            = "INVALID_HEADER"
	LocalErrorInvalidBodyFile          = "INVALID_BODY_FILE"
	LocalErrorQueryFilter              = "QUERY_FILTER_ERROR"
	LocalErrorEndpointNotResolved      = "ENDPOINT_NOT_RESOLVED"
	LocalErrorOther                    = "OTHER_LOCAL_ERROR"
)

var safeAPIErrorCodePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type Event struct {
	Schema             string                 `json:"schema"`
	EventID            string                 `json:"event_id"`
	EventType          string                 `json:"event_type"`
	Timestamp          string                 `json:"timestamp"`
	Pipeline           string                 `json:"pipeline"`
	CommandType        string                 `json:"command_type,omitempty"`
	Command            string                 `json:"command"`
	Params             []string               `json:"params,omitempty"`
	Result             string                 `json:"result"`
	LocalErrorCode     string                 `json:"local_error_code,omitempty"`
	APIErrorCode       string                 `json:"api_error_code,omitempty"`
	HTTPStatusCode     int                    `json:"http_status_code,omitempty"`
	DurationMs         int64                  `json:"duration_ms"`
	Plugin             string                 `json:"plugin,omitempty"`
	CLIVersion         string                 `json:"cli_version"`
	OS                 string                 `json:"os"`
	Arch               string                 `json:"arch"`
	GoVersion          string                 `json:"go_version"`
	LinuxDistroName    string                 `json:"linux_distro_name,omitempty"`
	LinuxDistroID      string                 `json:"linux_distro_id,omitempty"`
	LinuxDistroVersion string                 `json:"linux_distro_version,omitempty"`
	EnvironmentNames   []string               `json:"environment_names,omitempty"`
	Mode               string                 `json:"mode"`
	AgentName          string                 `json:"agent_name,omitempty"`
	InstanceID         string                 `json:"instance_id"`
	PluginAction       *PluginManagementEvent `json:"plugin_action,omitempty"`
}

type BuildInput struct {
	Scope        Scope
	RootArgs     []string
	InstanceID   string
	DurationMs   int64
	Err          error
	ExitCode     *int
	AIMode       bool
	PluginAction *PluginManagementEvent
}

type localErrorCoder interface {
	LocalErrorCode() string
}

func normalizedLocalErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var coded localErrorCoder
	if errors.As(err, &coded) {
		switch code := coded.LocalErrorCode(); code {
		case LocalErrorInvalidCommand,
			LocalErrorInvalidFlag,
			LocalErrorMissingRequiredParameter,
			LocalErrorInvalidArgument,
			LocalErrorInvalidOptionCombination,
			LocalErrorHelpOption,
			LocalErrorInvalidHeader,
			LocalErrorInvalidBodyFile,
			LocalErrorQueryFilter,
			LocalErrorEndpointNotResolved:
			return code
		}
	}
	if cli.IsAIRecoveryEligible(err) {
		return LocalErrorOther
	}
	return ""
}

func apiErrorDetails(err error) (string, int) {
	if err == nil {
		return "", 0
	}
	var teaErr *tea.SDKError
	if errors.As(err, &teaErr) {
		return safeAPIErrorCode(tea.StringValue(teaErr.Code)), safeHTTPStatus(tea.IntValue(teaErr.StatusCode))
	}
	var serverErr *sdkerrors.ServerError
	if errors.As(err, &serverErr) {
		return safeAPIErrorCode(serverErr.ErrorCode()), safeHTTPStatus(serverErr.HttpStatus())
	}
	return "", 0
}

func safeAPIErrorCode(code string) string {
	code = strings.TrimSpace(code)
	if !safeAPIErrorCodePattern.MatchString(code) {
		return ""
	}
	return code
}

func safeHTTPStatus(status int) int {
	if status < 100 || status > 599 {
		return 0
	}
	return status
}

func classifyResult(err error, exitCode *int) string {
	if err == nil && (exitCode == nil || *exitCode == 0) {
		return "Success"
	}
	if exitCode != nil && *exitCode != 0 {
		return "Failure"
	}
	if normalizedLocalErrorCode(err) != "" {
		return "UserFault"
	}
	return "Failure"
}

func BuildEvent(in BuildInput) (Event, error) {
	eventID, err := newUUID()
	if err != nil {
		return Event{}, err
	}

	pos := commandParts(in.RootArgs, in.Scope.CommandToken)
	command := normalizeCommand(pos)

	mode := "default"
	if in.AIMode {
		mode = "ai"
	}

	pluginField := ""
	if in.Scope.PluginName != "" {
		ver := in.Scope.PluginVer
		if ver == "" {
			ver = "unknown"
		}
		pluginField = in.Scope.PluginName + "@" + ver
	}
	eventType := EventTypeCommandCompleted
	if len(in.RootArgs) > 0 && strings.EqualFold(strings.TrimSpace(in.RootArgs[0]), "configure") {
		eventType = EventTypeConfigurationCompleted
	}
	if in.PluginAction != nil {
		eventType = EventTypePluginManagementCompleted
	}
	localErrorCode := normalizedLocalErrorCode(in.Err)
	apiErrorCode, httpStatusCode := apiErrorDetails(in.Err)
	linuxDistroName, linuxDistroID, linuxDistroVersion := detectLinuxDistro()

	return Event{
		Schema:             schemaVersion,
		EventID:            eventID,
		EventType:          eventType,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		Pipeline:           in.Scope.Pipeline,
		CommandType:        DetectCommandType(in.RootArgs),
		Command:            command,
		Params:             ExtractParamNames(in.RootArgs),
		Result:             classifyResult(in.Err, in.ExitCode),
		LocalErrorCode:     localErrorCode,
		APIErrorCode:       apiErrorCode,
		HTTPStatusCode:     httpStatusCode,
		DurationMs:         in.DurationMs,
		Plugin:             pluginField,
		CLIVersion:         cli.GetVersion(),
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		GoVersion:          runtime.Version(),
		LinuxDistroName:    linuxDistroName,
		LinuxDistroID:      linuxDistroID,
		LinuxDistroVersion: linuxDistroVersion,
		EnvironmentNames:   alibabaCloudEnvironmentNames(),
		Mode:               mode,
		AgentName:          util.DetectAgentName(),
		InstanceID:         in.InstanceID,
		PluginAction:       in.PluginAction,
	}, nil
}
