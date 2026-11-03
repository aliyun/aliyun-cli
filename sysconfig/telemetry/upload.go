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
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultTelemetryEndpoint = "https://openapi-acore.aliyuncs.com/v1/telemetry/events"

type uploadEvent struct {
	SchemaVersion string         `json:"schemaVersion"`
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	Timestamp     string         `json:"timestamp"`
	ClientName    string         `json:"clientName"`
	ClientVersion string         `json:"clientVersion"`
	InstanceID    string         `json:"instanceId"`
	Status        string         `json:"status"`
	DurationMs    int64          `json:"durationMs"`
	Attributes    map[string]any `json:"attributes"`
}

type uploadBatch struct {
	Events []uploadEvent `json:"events"`
}

func telemetryUploadEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT")); v != "" {
		return v
	}
	return defaultTelemetryEndpoint
}

func UploadFile(configDir, pipeline, cacheFile string) error {
	return uploadFile(configDir, pipeline, cacheFile, "")
}

func uploadFile(configDir, pipeline, cacheFile, eventID string) error {
	record := diagnosticRecord{EventID: eventID, Pipeline: pipeline}
	if pipeline != PipelinePublic && pipeline != PipelineInner {
		record.Stage, record.Status, record.Reason = "upload", "skipped", "invalid_pipeline"
		writeDiagnostic(configDir, record)
		return nil
	}
	cacheDir := GetCacheDir(configDir, pipeline)
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		record.Stage, record.Status, record.Reason = "cache_read", "skipped", "invalid_cache_path"
		record.ErrorKind = diagnosticErrorKind(err)
		writeDiagnostic(configDir, record)
		return nil
	}
	absCacheFile, err := filepath.Abs(cacheFile)
	if err != nil ||
		filepath.Dir(absCacheFile) != absCacheDir ||
		!strings.HasSuffix(absCacheFile, ".ndjson") {
		record.Stage, record.Status, record.Reason = "cache_read", "skipped", "invalid_cache_path"
		record.ErrorKind = diagnosticErrorKind(err)
		writeDiagnostic(configDir, record)
		return nil
	}
	record.Stage, record.Status = "cache_read", "started"
	writeDiagnostic(configDir, record)
	lines, readErr := readNDJSON(absCacheFile)
	record.Status = "succeeded"
	if readErr != nil {
		record.Status, record.ErrorKind = "failed", diagnosticErrorKind(readErr)
	}
	writeDiagnostic(configDir, record)
	removeUploadCache(configDir, pipeline, absCacheFile, eventID)
	if readErr != nil {
		return nil
	}
	if len(lines) == 0 {
		record.Stage, record.Status, record.Reason = "upload", "skipped", "cache_empty"
		writeDiagnostic(configDir, record)
		return nil
	}

	var event Event
	if err := json.Unmarshal(lines[0], &event); err != nil {
		record.Stage, record.Status, record.Reason = "upload", "skipped", "invalid_event"
		record.ErrorKind = diagnosticErrorKind(err)
		writeDiagnostic(configDir, record)
		return nil
	}
	record.EventID = event.EventID
	if event.Pipeline != pipeline {
		record.Stage, record.Status, record.Reason = "upload", "skipped", "pipeline_mismatch"
		writeDiagnostic(configDir, record)
		return nil
	}
	record.Stage, record.Status = "upload", "started"
	writeDiagnostic(configDir, record)
	result := postTelemetryBatchResult([]uploadEvent{newUploadEvent(event)}, nil)
	record.Status, record.Reason, record.ErrorKind = "failed", result.reason, result.errorKind
	if result.success {
		record.Status = "succeeded"
	}
	record.HTTPStatus, record.DurationMs = result.httpStatus, result.durationMs
	writeDiagnostic(configDir, record)
	return nil
}

func removeUploadCache(configDir, pipeline, cacheFile, eventID string) {
	record := diagnosticRecord{Stage: "cache_remove", Status: "succeeded", EventID: eventID, Pipeline: pipeline}
	if err := os.Remove(cacheFile); err != nil {
		record.Status, record.ErrorKind = "failed", diagnosticErrorKind(err)
	}
	writeDiagnostic(configDir, record)
}

type telemetryUploadResult struct {
	success    bool
	httpStatus int
	reason     string
	errorKind  string
	durationMs int64
}

func postTelemetryBatch(batch []uploadEvent) bool {
	return postTelemetryBatchResult(batch, nil).success
}

func postTelemetryBatchResult(batch []uploadEvent, client *http.Client) (result telemetryUploadResult) {
	started := time.Now()
	defer func() { result.durationMs = time.Since(started).Milliseconds() }()
	body, err := json.Marshal(uploadBatch{Events: batch})
	if err != nil {
		result.reason, result.errorKind = "marshal_failed", diagnosticErrorKind(err)
		return result
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequest(http.MethodPost, telemetryUploadEndpoint(), bytes.NewReader(body))
	if err != nil {
		result.reason, result.errorKind = "request_invalid", diagnosticErrorKind(err)
		return result
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		result.errorKind = diagnosticErrorKind(err)
		return result
	}
	defer resp.Body.Close()
	result.httpStatus = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.reason = "http_error"
		return result
	}
	result.success = true
	return result
}

func newUploadEvent(event Event) uploadEvent {
	attributes := map[string]any{
		"command":       event.Command,
		"os":            event.OS,
		"arch":          event.Arch,
		"goVersion":     event.GoVersion,
		"aiModeEnabled": event.AIModeEnabled,
	}
	// Only emit pipeline for inner events; public is the default and omitted.
	if event.Pipeline == PipelineInner {
		attributes["pipeline"] = PipelineInner
	}
	addAttribute(attributes, "commandType", event.CommandType)
	addAttribute(attributes, "openapiRuntimeVersion", event.OpenAPIRuntimeVersion)
	if len(event.Params) > 0 {
		attributes["parameterNames"] = event.Params
	}
	addAttribute(attributes, "localErrorCode", event.LocalErrorCode)
	addAttribute(attributes, "apiErrorCode", event.APIErrorCode)
	if event.HTTPStatusCode != 0 {
		attributes["httpStatusCode"] = event.HTTPStatusCode
	}
	addAttribute(attributes, "linuxDistroName", event.LinuxDistroName)
	addAttribute(attributes, "linuxDistroId", event.LinuxDistroID)
	addAttribute(attributes, "linuxDistroVersion", event.LinuxDistroVersion)
	if len(event.EnvironmentNames) > 0 {
		attributes["environmentVariableNames"] = event.EnvironmentNames
	}
	addAttribute(attributes, "shellType", event.ShellType)
	attributes["stdinIsTTY"] = event.StdinIsTTY
	addAttribute(attributes, "locale", event.Locale)
	addAttribute(attributes, "authMode", event.AuthMode)
	addAttribute(attributes, "authModeSource", event.AuthModeSource)
	addAttribute(attributes, "plugin", event.Plugin)
	addAttribute(attributes, "agentName", event.AgentName)
	if action := event.PluginAction; action != nil {
		attributes["operation"] = action.Operation
		if len(action.PluginNames) > 0 {
			attributes["pluginNames"] = action.PluginNames
		}
		addAttribute(attributes, "fromVersion", action.FromVersion)
		addAttribute(attributes, "toVersion", action.ToVersion)
		addAttribute(attributes, "pluginType", action.PluginType)
		if action.Inner != nil {
			attributes["inner"] = *action.Inner
		}
		addAttribute(attributes, "sourceType", action.SourceType)
	}
	eventType := event.EventType
	if eventType == "" {
		eventType = EventTypeCommandCompleted
	}
	return uploadEvent{
		SchemaVersion: event.Schema,
		EventID:       event.EventID,
		EventType:     eventType,
		Timestamp:     event.Timestamp,
		ClientName:    "aliyun-cli",
		ClientVersion: event.CLIVersion,
		InstanceID:    event.InstanceID,
		Status:        uploadStatus(event.Result),
		DurationMs:    event.DurationMs,
		Attributes:    attributes,
	}
}

func addAttribute(attributes map[string]any, name, value string) {
	if value != "" {
		attributes[name] = value
	}
}

func uploadStatus(result string) string {
	switch result {
	case "Success":
		return "success"
	case "UserFault":
		return "user_fault"
	default:
		return "failure"
	}
}

func readNDJSON(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []json.RawMessage
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		out = append(out, append(json.RawMessage(nil), line...))
	}
	return out, sc.Err()
}

func spawnUpload(configDir, pipeline, cacheFile, eventID string) {
	record := diagnosticRecord{Stage: "upload_spawn", Status: "started", EventID: eventID, Pipeline: pipeline}
	writeDiagnostic(configDir, record)
	exe, err := os.Executable()
	if err != nil {
		record.Status, record.Reason, record.ErrorKind = "failed", "executable_unavailable", diagnosticErrorKind(err)
		writeDiagnostic(configDir, record)
		removeUploadCache(configDir, pipeline, cacheFile, eventID)
		return
	}
	cmd := exec.Command(
		exe,
		"__telemetry-upload",
		"--pipeline",
		pipeline,
		"--config-dir",
		configDir,
		"--cache-file",
		cacheFile,
		"--event-id",
		eventID,
	)
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		record.Status, record.Reason, record.ErrorKind = "failed", "process_start_failed", diagnosticErrorKind(err)
		writeDiagnostic(configDir, record)
		removeUploadCache(configDir, pipeline, cacheFile, eventID)
		return
	}
	record.Status = "succeeded"
	writeDiagnostic(configDir, record)
}

// RunUploadCommand handles the hidden __telemetry-upload entrypoint.
func RunUploadCommand(args []string) int {
	configDir := ""
	cacheFile := ""
	pipeline := ""
	eventID := ""
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--config-dir="):
			configDir = strings.TrimPrefix(args[i], "--config-dir=")
		case args[i] == "--config-dir":
			if i+1 < len(args) {
				configDir = args[i+1]
				i++
			}
		case strings.HasPrefix(args[i], "--pipeline="):
			pipeline = strings.TrimPrefix(args[i], "--pipeline=")
		case args[i] == "--pipeline":
			if i+1 < len(args) {
				pipeline = args[i+1]
				i++
			}
		case strings.HasPrefix(args[i], "--cache-file="):
			cacheFile = strings.TrimPrefix(args[i], "--cache-file=")
		case args[i] == "--cache-file":
			if i+1 < len(args) {
				cacheFile = args[i+1]
				i++
			}
		case strings.HasPrefix(args[i], "--event-id="):
			eventID = strings.TrimPrefix(args[i], "--event-id=")
		case args[i] == "--event-id":
			if i+1 < len(args) {
				eventID = args[i+1]
				i++
			}
		}
	}
	if configDir == "" || pipeline == "" || cacheFile == "" {
		writeDiagnostic(configDir, diagnosticRecord{Stage: "upload", Status: "skipped", Reason: "missing_arguments", EventID: eventID, Pipeline: pipeline})
		return 0
	}
	if pipeline != PipelineInner && pipeline != PipelinePublic {
		writeDiagnostic(configDir, diagnosticRecord{Stage: "upload", Status: "skipped", Reason: "invalid_pipeline", EventID: eventID})
		return 0
	}
	_ = uploadFile(configDir, pipeline, cacheFile, eventID)
	return 0
}
