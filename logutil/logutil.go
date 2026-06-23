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

package logutil

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
	Fatal
)

var (
	currentLevel = Error
	output       io.Writer = os.Stderr
)

func SetLevel(level Level) {
	currentLevel = level
}

func SetOutput(w io.Writer) {
	if w != nil {
		output = w
	}
}

func IsInfoEnabled() bool {
	return currentLevel <= Info
}

func LogThrottlingRetry(format string, args ...interface{}) {
	if currentLevel > Info {
		return
	}
	message := fmt.Sprintf(format, args...)
	output.Write([]byte("aliyun: " + message + "\n"))
}

func InitFromContext(logLevel string) {
	if level, err := ParseLevel(logLevel); err == nil {
		SetLevel(level)
		return
	}
	if env := strings.TrimSpace(os.Getenv("ALIBABA_CLOUD_CLI_LOG_CONFIG")); env != "" {
		if level, err := ParseLevel(env); err == nil {
			SetLevel(level)
		}
	}
}

func ParseLevel(level string) (Level, error) {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return Debug, nil
	case "INFO":
		return Info, nil
	case "WARN", "WARNING":
		return Warn, nil
	case "ERROR":
		return Error, nil
	case "FATAL":
		return Fatal, nil
	default:
		return Error, fmt.Errorf("invalid log level: %s", level)
	}
}
