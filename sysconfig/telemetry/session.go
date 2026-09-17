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
	"sync"
	"time"
)

type activeSession struct {
	configDir string
	scope     InnerScope
	profile   ProfileSnapshot
	rootArgs  []string
	start     time.Time
	aiMode    bool
	cfg       *Config
}

var (
	sessionMu sync.Mutex
	active    *activeSession
	ended     bool
)

func resetSessionState() {
	sessionMu.Lock()
	active = nil
	ended = false
	sessionMu.Unlock()
}

func beginSession(configDir string, scope InnerScope, profile ProfileSnapshot, rootArgs []string, aiMode bool, cfg *Config) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	active = &activeSession{
		configDir: configDir,
		scope:     scope,
		profile:   profile,
		rootArgs:  append([]string(nil), rootArgs...),
		start:     time.Now(),
		aiMode:    aiMode,
		cfg:       cfg,
	}
	ended = false
}

func takeSessionForEnd() (*activeSession, bool) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if active == nil || ended {
		return nil, false
	}
	ended = true
	return active, true
}

func sessionDurationMs(s *activeSession) int64 {
	if s == nil {
		return 0
	}
	return time.Since(s.start).Milliseconds()
}
