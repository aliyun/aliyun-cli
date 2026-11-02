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
// When present (including empty), it overrides profile signature_version but
// loses to an assigned --cli-sign-version.
const EnvSignatureVersion = "ALIBABA_CLOUD_SIGNATURE_VERSION"

// SignatureVersionV4 enables ACS4-HMAC-SHA256 via SignatureVersion = "v4".
// Empty means keep the default signing path. Other values are rejected.
const SignatureVersionV4 = "v4"

// NormalizeSignatureVersion validates and canonicalizes a signature version.
// Empty clears / keeps the default signing path. Only "v4" is supported now.
func NormalizeSignatureVersion(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	if strings.EqualFold(v, SignatureVersionV4) {
		return SignatureVersionV4, nil
	}
	return "", fmt.Errorf("unsupported signature_version %q; only %q is supported (set empty to clear and use default)",
		v, SignatureVersionV4)
}

// CliSignVersionFromFlags returns the --cli-sign-version value and whether the
// flag was assigned. An assigned empty value clears the effective version for
// this invocation (overrides env / profile).
func CliSignVersionFromFlags(fs *cli.FlagSet) (value string, assigned bool) {
	if fs == nil {
		return "", false
	}
	f := CliSignVersionFlag(fs)
	if f == nil {
		return "", false
	}
	v, ok := f.GetValue()
	if !ok {
		return "", false
	}
	return strings.TrimSpace(v), true
}

// EffectiveSignatureVersion returns the active signature version string (unvalidated).
// Precedence: assigned --cli-sign-version (including empty) >
// present ALIBABA_CLOUD_SIGNATURE_VERSION (including empty) >
// profile.signature_version.
func (cp *Profile) EffectiveSignatureVersion(cliSignVersion string, cliAssigned bool) string {
	if cliAssigned {
		return strings.TrimSpace(cliSignVersion)
	}
	if v, ok := os.LookupEnv(EnvSignatureVersion); ok {
		return strings.TrimSpace(v)
	}
	if cp == nil {
		return ""
	}
	return strings.TrimSpace(cp.SignatureVersion)
}

// ResolvedSignatureVersion returns the normalized active signature version.
func (cp *Profile) ResolvedSignatureVersion(cliSignVersion string, cliAssigned bool) (string, error) {
	return NormalizeSignatureVersion(cp.EffectiveSignatureVersion(cliSignVersion, cliAssigned))
}

// UseSignatureV4 reports whether OpenAPI calls should use signature V4.
// Empty keeps the default signing path. Unsupported values return an error.
func (cp *Profile) UseSignatureV4(cliSignVersion string, cliAssigned bool) (bool, error) {
	v, err := cp.ResolvedSignatureVersion(cliSignVersion, cliAssigned)
	if err != nil {
		return false, err
	}
	return v == SignatureVersionV4, nil
}
