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

package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-openapi-runtime/redact"
)

// LogLevel controls which diagnostic messages are emitted to stderr.
// Ordering matches aliyun-cli-runtime/http: lower = more verbose.
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

var levelNames = map[LogLevel]string{
	LogDebug: "DEBUG",
	LogInfo:  "INFO",
	LogWarn:  "WARN",
	LogError: "ERROR",
	LogFatal: "FATAL",
}

var levelColors = map[LogLevel]string{
	LogDebug: "\033[36m",
	LogInfo:  "\033[32m",
	LogWarn:  "\033[33m",
	LogError: "\033[31m",
	LogFatal: "\033[35m",
}

const colorReset = "\033[0m"

// envLogConfig is the plugin-compatible override for log presets.
const envLogConfig = "ALIBABA_CLOUD_CLI_LOG_CONFIG"

type logConfig struct {
	Level       LogLevel
	EnableTime  bool
	EnableColor bool
	Output      io.Writer
}

var (
	productionConfig   = logConfig{Level: LogError, Output: os.Stderr}
	developmentConfig  = logConfig{Level: LogInfo, EnableTime: true, EnableColor: true, Output: os.Stderr}
	debugConfig        = logConfig{Level: LogDebug, EnableTime: true, EnableColor: true, Output: os.Stderr}
	quietConfig        = logConfig{Level: LogFatal, Output: os.Stderr}
	ciConfig           = logConfig{Level: LogWarn, EnableTime: true, Output: os.Stderr}
)

type logger struct {
	mu          sync.Mutex
	level       LogLevel
	output      io.Writer
	enableTime  bool
	enableColor bool
}

var globalLogger = &logger{
	level:       LogError,
	output:      os.Stderr,
	enableColor: true,
}

// InitLogger applies ALIBABA_CLOUD_CLI_LOG_CONFIG / --log-level, matching
// aliyun-cli-runtime InitLogger. Dry-run short-circuits (plugin parity:
// dry-run has its own dump path and does not change the log level).
func InitLogger(logLevel string, dryRun bool) {
	if dryRun {
		return
	}
	if env := strings.TrimSpace(os.Getenv(envLogConfig)); env != "" {
		if !applyNamedConfig(env) {
			Warn("Invalid log config: %s, using default (production)", env)
		}
		return
	}
	if logLevel == "" {
		return
	}
	if !applyNamedConfig(logLevel) {
		Warn("Invalid log config: %s, using default (production)", logLevel)
	}
}

func applyNamedConfig(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "production", "prod":
		applyConfig(productionConfig)
		return true
	case "development", "dev", "info":
		applyConfig(developmentConfig)
		return true
	case "debug", "verbose":
		applyConfig(debugConfig)
		return true
	case "quiet":
		applyConfig(quietConfig)
		return true
	case "ci":
		applyConfig(ciConfig)
		return true
	}

	// Help text advertises DEBUG/INFO/WARN/ERROR; map those names onto
	// the same presets the plugin aliases use.
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "DEBUG":
		applyConfig(debugConfig)
		return true
	case "INFO":
		applyConfig(developmentConfig)
		return true
	case "WARN", "WARNING":
		applyConfig(ciConfig)
		return true
	case "ERROR":
		applyConfig(productionConfig)
		return true
	case "FATAL":
		applyConfig(quietConfig)
		return true
	default:
		return false
	}
}

func applyConfig(c logConfig) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	globalLogger.level = c.Level
	globalLogger.enableTime = c.EnableTime
	globalLogger.enableColor = c.EnableColor
	if c.Output != nil {
		globalLogger.output = c.Output
	}
}

// ResetLoggerForTest restores the default ERROR logger (tests only).
func ResetLoggerForTest() {
	applyConfig(logConfig{Level: LogError, EnableColor: true, Output: os.Stderr})
}

// SetLoggerOutputForTest redirects log output (tests only).
func SetLoggerOutputForTest(w io.Writer) {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	if w != nil {
		globalLogger.output = w
	}
}

// IsDebugEnabled reports whether DEBUG logs are active.
func IsDebugEnabled() bool {
	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()
	return globalLogger.level <= LogDebug
}

func Debug(format string, args ...any) { globalLogger.log(LogDebug, format, args...) }
func Info(format string, args ...any)  { globalLogger.log(LogInfo, format, args...) }
func Warn(format string, args ...any)  { globalLogger.log(LogWarn, format, args...) }

func (l *logger) log(level LogLevel, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level < l.level {
		return
	}
	var sb strings.Builder
	if l.enableTime {
		sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
		sb.WriteString(" ")
	}
	name := levelNames[level]
	if l.enableColor {
		sb.WriteString(levelColors[level])
		sb.WriteString(fmt.Sprintf("[%-5s]", name))
		sb.WriteString(colorReset)
	} else {
		sb.WriteString(fmt.Sprintf("[%-5s]", name))
	}
	sb.WriteString(" ")
	sb.WriteString(fmt.Sprintf(format, args...))
	sb.WriteString("\n")
	_, _ = l.output.Write([]byte(sb.String()))
}

func logSection(title string) {
	if !IsDebugEnabled() {
		return
	}
	sep := strings.Repeat("=", 60)
	Debug("%s", sep)
	Debug("%s", title)
	Debug("%s", sep)
}

// LogArgs dumps the parsed API argument map at DEBUG (masked).
// Matches the Go plugin: compact single-line JSON (not pretty-printed;
// request Body uses logJSON for indentation).
func LogArgs(args map[string]any) {
	if !IsDebugEnabled() {
		return
	}
	logSection("Execute Arguments")
	if len(args) == 0 {
		Debug("Arguments: (empty)")
		return
	}
	masked := make(map[string]any, len(args))
	for k, v := range args {
		if redact.IsSensitive(k) {
			if s, ok := v.(string); ok {
				masked[k] = redact.MaskValue(s)
			} else {
				masked[k] = "***"
			}
			continue
		}
		masked[k] = v
	}
	if b, err := json.Marshal(masked); err == nil {
		Debug("Arguments: %s", string(b))
	} else {
		Debug("Arguments: (error marshaling: %v)", err)
	}
}

// logJSON pretty-prints value at DEBUG, matching aliyun-cli-runtime's
// LogJSON: MarshalIndent with two spaces, each line a separate Debug.
func logJSON(label string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		Debug("%s: (failed to marshal: %v)", label, err)
		return
	}
	Debug("%s:", label)
	for _, line := range strings.Split(string(b), "\n") {
		if line != "" {
			Debug("  %s", line)
		}
	}
}

// LogRequest dumps the outbound HTTP request at DEBUG (masked).
func LogRequest(req *AssembledRequest) {
	if !IsDebugEnabled() || req == nil {
		return
	}
	logSection("HTTP Request")
	Debug("Method: %s", req.Method)
	Debug("URL: %s", req.Pathname)
	Debug("API Version: %s", req.Version)
	Debug("API Action: %s", req.Action)
	Debug("Protocol: %s", req.Protocol)
	Debug("Style: %s", req.Style)
	if req.Endpoint != "" {
		Debug("Endpoint: %s", req.Endpoint)
	}
	logStringMap("Headers", req.Headers)
	logStringMap("Query Parameters", req.Query)
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			Debug("Body (string): %s", v)
		case []byte:
			Debug("Body (bytes): %s", string(v))
		default:
			logJSON("Body", redact.MaskAny(req.Body))
		}
	}
}

// LogResponse dumps the inbound HTTP response at DEBUG (masked).
func LogResponse(resp *Response) {
	if !IsDebugEnabled() || resp == nil {
		return
	}
	logSection("HTTP Response")
	Debug("Status Code: %d", resp.StatusCode)
	if len(resp.Headers) > 0 {
		Debug("Response Headers:")
		keys := make([]string, 0, len(resp.Headers))
		for k := range resp.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range resp.Headers[k] {
				Debug("  %s: %s", k, redact.MaskKV(k, v))
			}
		}
	}
	body := string(resp.Raw)
	if body == "" {
		Debug("Response Body: (empty)")
		return
	}
	if resp.Parsed != nil {
		if b, err := json.Marshal(redact.MaskAny(resp.Parsed)); err == nil {
			Debug("Response Body: %s", string(b))
			return
		}
	}
	if len(body) > 1000 {
		Debug("Response Body (truncated): %s... (%d bytes total)", body[:1000], len(body))
		return
	}
	Debug("Response Body: %s", body)
}

func logStringMap(title string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	Debug("%s:", title)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		Debug("  %s: %s", k, redact.MaskKV(k, m[k]))
	}
}
