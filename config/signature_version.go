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

// Supported OpenAPI signature_version values for --cli-sign-version / env / profile.
const (
	// SignatureVersionV2 maps to conf.SignatureAlgorithm = "v2" (legacy RPC signing).
	SignatureVersionV2 = "v2"
	// SignatureVersionV3 is the default ACS3 path; no SignatureVersion/SignatureAlgorithm override.
	SignatureVersionV3 = "v3"
	// SignatureVersionV4 enables ACS4-HMAC-SHA256 via SignatureVersion = "v4".
	SignatureVersionV4 = "v4"
)

// NormalizeSignatureVersion validates and canonicalizes a signature version.
// Empty means unset (same runtime behavior as v3). Supported: v2, v3, v4.
func NormalizeSignatureVersion(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	switch strings.ToLower(v) {
	case SignatureVersionV2, SignatureVersionV3, SignatureVersionV4:
		return strings.ToLower(v), nil
	default:
		return "", fmt.Errorf("unsupported signature_version %q; supported values: %q, %q, %q",
			v, SignatureVersionV2, SignatureVersionV3, SignatureVersionV4)
	}
}

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

// EffectiveSignatureVersion returns the active signature version string (unvalidated).
// Precedence: --cli-sign-version > ALIBABA_CLOUD_SIGNATURE_VERSION > profile.signature_version.
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

// ResolvedSignatureVersion returns the normalized active signature version.
func (cp *Profile) ResolvedSignatureVersion(cliSignVersion string) (string, error) {
	return NormalizeSignatureVersion(cp.EffectiveSignatureVersion(cliSignVersion))
}

// UseSignatureV4 reports whether OpenAPI calls should use signature V4.
// Empty / v3 keep the default ACS3 path; v2 selects legacy SignatureAlgorithm.
// Unsupported values return an error so callers do not silently fall back.
func (cp *Profile) UseSignatureV4(cliSignVersion string) (bool, error) {
	v, err := cp.ResolvedSignatureVersion(cliSignVersion)
	if err != nil {
		return false, err
	}
	return v == SignatureVersionV4, nil
}
