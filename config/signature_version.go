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
// After Profile.OverwriteWithFlags, callers should read profile.SignatureVersion
// directly; the env is only a fallback when profile and flags are unset.
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
// flag was assigned. An assigned empty value clears the effective version.
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

// ResolveSignatureVersion applies precedence: command flag (including empty) >
// profile value > environment variable > default (empty).
// profileValue is the signature_version already loaded from the profile file
// (before flag/env overlay). Pass "" when no profile is available.
func ResolveSignatureVersion(cliValue string, cliAssigned bool, profileValue string) (string, error) {
	var raw string
	switch {
	case cliAssigned:
		raw = cliValue
	case strings.TrimSpace(profileValue) != "":
		raw = profileValue
	default:
		if v, ok := os.LookupEnv(EnvSignatureVersion); ok {
			raw = v
		}
	}
	return NormalizeSignatureVersion(raw)
}

// applySignatureVersionFromFlags overlays --cli-sign-version / env onto the
// profile field using Flag > Profile > Env precedence. An assigned empty flag
// clears the value and is not backfilled by the environment.
func (cp *Profile) applySignatureVersionFromFlags(ctx *cli.Context) {
	if cp == nil {
		return
	}
	cliValue, cliAssigned := "", false
	if ctx != nil {
		cliValue, cliAssigned = CliSignVersionFromFlags(ctx.Flags())
	}
	resolved, err := ResolveSignatureVersion(cliValue, cliAssigned, cp.SignatureVersion)
	if err == nil {
		cp.SignatureVersion = resolved
		return
	}
	// Keep the raw selected value so Validate can surface the error.
	switch {
	case cliAssigned:
		cp.SignatureVersion = strings.TrimSpace(cliValue)
	case strings.TrimSpace(cp.SignatureVersion) != "":
		cp.SignatureVersion = strings.TrimSpace(cp.SignatureVersion)
	default:
		if v, ok := os.LookupEnv(EnvSignatureVersion); ok {
			cp.SignatureVersion = strings.TrimSpace(v)
		}
	}
}

// ValidateSignatureVersion validates profile.SignatureVersion after overlays.
func (cp *Profile) ValidateSignatureVersion() error {
	if cp == nil {
		return nil
	}
	normalized, err := NormalizeSignatureVersion(cp.SignatureVersion)
	if err != nil {
		return err
	}
	cp.SignatureVersion = normalized
	return nil
}

// UseSignatureV4 reports whether the already-resolved profile.SignatureVersion
// selects signature V4. Call after OverwriteWithFlags.
func (cp *Profile) UseSignatureV4() (bool, error) {
	if err := cp.ValidateSignatureVersion(); err != nil {
		return false, err
	}
	if cp == nil {
		return false, nil
	}
	return cp.SignatureVersion == SignatureVersionV4, nil
}
