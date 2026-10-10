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
	"path/filepath"
	"strings"
	"time"
)

const defaultTelemetryEndpoint = "https://telemetry-inner.aliyun-inc.com/v1/telemetry/events"

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

func telemetryUploadEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("ALIYUN_CLI_TELEMETRY_ENDPOINT")); v != "" {
		return v
	}
	return defaultTelemetryEndpoint
}

func UploadFile(configDir, pipeline, cacheFile string) error {
	cacheDir := GetCacheDir(configDir, pipeline)
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return nil
	}
	absCacheFile, err := filepath.Abs(cacheFile)
	if err != nil ||
		filepath.Dir(absCacheFile) != absCacheDir ||
		!strings.HasSuffix(absCacheFile, ".ndjson") {
		return nil
	}
	lines, readErr := readNDJSON(absCacheFile)
	_ = os.Remove(absCacheFile)
	if readErr != nil || len(lines) == 0 {
		return nil
	}

	var event Event
	if err := json.Unmarshal(lines[0], &event); err != nil {
		return nil
	}
	if event.Pipeline != pipeline {
		return nil
	}
	if !postTelemetryBatch([]uploadEvent{newUploadEvent(event)}) {
		return nil
	}
	cfg, _ := Load(configDir)
	return MarkUploaded(configDir, cfg, pipeline)
}

func postTelemetryBatch(batch []uploadEvent) bool {
	body, err := json.Marshal(batch)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, telemetryUploadEndpoint(), bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	return true
}

func newUploadEvent(event Event) uploadEvent {
	attributes := map[string]any{
		"pipeline":  event.Pipeline,
		"command":   event.Command,
		"os":        event.OS,
		"arch":      event.Arch,
		"goVersion": event.GoVersion,
		"mode":      event.Mode,
	}
	addAttribute(attributes, "innerTrigger", event.InnerTrigger)
	if len(event.Params) > 0 {
		attributes["parameterNames"] = event.Params
	}
	addAttribute(attributes, "errorType", event.ErrorType)
	addAttribute(attributes, "plugin", event.Plugin)
	addAttribute(attributes, "regionId", event.RegionID)
	addAttribute(attributes, "agentName", event.AgentName)
	addAttribute(attributes, "vendor", event.Vendor)
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
		eventType = "cli.command.completed"
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
