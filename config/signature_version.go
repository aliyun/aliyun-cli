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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

// EnvSignatureVersion selects the OpenAPI request signature version.
// When set, it overrides profile signature_version but loses to --cli-sign-version.
const EnvSignatureVersion = "ALIBABA_CLOUD_SIGNATURE_VERSION"

// SignatureVersionV4 is the only non-default signature version supported in phase 1.
const SignatureVersionV4 = "v4"

// CliSignVersionFromFlags returns the --cli-sign-version value for this invocation.
// Empty means the flag was not set.
func CliSignVersionFromFlags(fs *cli.FlagSet) string {
	if fs == nil {
		return ""
	}
	f := CliSignVersionFlag(fs)
	if f == nil {
		return ""
	}
	v, ok := f.GetValue()
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// EffectiveSignatureVersion returns the active signature version.
// Precedence matches the SLS plugin hook: --cli-sign-version >
// ALIBABA_CLOUD_SIGNATURE_VERSION > profile.signature_version.
func (cp *Profile) EffectiveSignatureVersion(cliSignVersion string) string {
	if v := strings.TrimSpace(cliSignVersion); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(EnvSignatureVersion)); v != "" {
		return v
	}
	if cp == nil {
		return ""
	}
	return strings.TrimSpace(cp.SignatureVersion)
}

// UseSignatureV4 reports whether OpenAPI calls should use signature V4.
// Empty means keep the current (ACS3 / legacy) behavior.
// Unsupported values return an error so callers do not silently fall back.
func (cp *Profile) UseSignatureV4(cliSignVersion string) (bool, error) {
	v := strings.TrimSpace(cp.EffectiveSignatureVersion(cliSignVersion))
	if v == "" {
		return false, nil
	}
	if strings.EqualFold(v, SignatureVersionV4) {
		return true, nil
	}
	return false, fmt.Errorf("unsupported signature_version %q; only %q is supported", v, SignatureVersionV4)
}
