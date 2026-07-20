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
	"time"

	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

// Host is the seam between the engine and its embedding CLI/app. It
// supplies everything the engine cannot know on its own — the default
// region, the caller's credential, and the profile-derived wire
// settings — WITHOUT the engine depending on aliyun-cli's config/
// profile machinery.
//
// The aliyun-cli main module provides a profile-backed implementation;
// tests and third-party embedders can supply their own. Splitting
// Region()/Settings() (cheap, secret-free) from Credential() lets
// dry-run resolve endpoint/timeouts without ever touching secrets.
type Host interface {
	// Region returns the default region (e.g. "cn-hangzhou") or ""
	// when none is configured. Must be cheap and secret-free.
	Region() string

	// Settings returns the profile-derived wire settings (timeouts,
	// retry, endpoint type, language). Must be cheap and secret-free.
	// This mirrors the environment the Go plugin path exports to a
	// plugin process, so the in-process engine behaves the same.
	Settings() Settings

	// Credential resolves the caller's credential. It is only invoked
	// on a real send (never on dry-run), so implementations may defer
	// expensive work (STS exchange, file IO) until here.
	Credential() (credentialsv2.Credential, error)
}

// Settings carries the profile-derived, non-secret runtime knobs the
// engine applies to a call. Zero values mean "unset — use the SDK
// default".
type Settings struct {
	ReadTimeout    time.Duration
	ConnectTimeout time.Duration
	RetryCount     int
	EndpointType   string // e.g. "vpc" -> prefer VPC endpoints
	Language       string // "zh" / "en"

	// SkipSecureVerify disables TLS certificate verification (from
	// host --skip-secure-verify). Not recommended.
	SkipSecureVerify bool

	// UserAgent is the final suffix/string the host wants appended on
	// the OpenAPI client (custom --user-agent plus AI-mode segments).
	// Empty means leave the SDK default.
	UserAgent string
}

// UseVPC reports whether VPC endpoints should be preferred.
func (s Settings) UseVPC() bool { return s.EndpointType == "vpc" }

// StaticHost is a trivial Host useful for tests and non-interactive
// embedders: fixed region/settings and an optional pre-resolved
// credential.
type StaticHost struct {
	RegionID    string
	SettingsVal Settings
	Cred        credentialsv2.Credential
	CredErr     error
}

// Region returns the configured region.
func (h StaticHost) Region() string { return h.RegionID }

// Settings returns the configured settings.
func (h StaticHost) Settings() Settings { return h.SettingsVal }

// Credential returns the configured credential or error.
func (h StaticHost) Credential() (credentialsv2.Credential, error) {
	return h.Cred, h.CredErr
}
