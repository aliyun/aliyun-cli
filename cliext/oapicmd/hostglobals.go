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

package oapicmd

import (
	"strings"
	"sync"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// The engine and the host register their global flags independently.
// The host owns profile / credential / network flags (they resolve
// credentials the engine consumes via runtime.Host) plus a small set of
// transport flags (--user-agent, AI-mode) that are folded into
// runtime.Settings; the engine owns its runtime flags (--cli-dry-run,
// --cli-query, --pager, ...).
//
// The top-level product router forwards the API argv tail without parsing its
// product-specific flags, so host global flags placed after the command would
// otherwise reach the engine's argparser and be rejected as unknown.
// extractHostGlobals pulls them out of the tail, applies them to the context
// (so config.LoadProfileWithContext sees them), and returns the remaining args
// for the engine.

var (
	hostGlobalsOnce sync.Once
	hostGlobalsRef  *cli.FlagSet
)

// engineOwnedFlags are config-registered flag names that the ENGINE
// handles itself (--region, --endpoint) or that collide with an API's
// own parameter (RegionId → APIs expose their own --region-id). These
// are left in the tail for the engine / argparser.
var engineOwnedFlags = map[string]bool{
	config.RegionFlagName:   true,
	config.EndpointFlagName: true,
	config.RegionIdFlagName: true,
}

// callerTransportFlags are host-owned flags that are NOT in
// config.AddFlags but must be stripped from the engine argv and
// applied via runtime.Settings (UA / AI mode).
var callerTransportFlags = map[string]cli.AssignedMode{
	"user-agent":        cli.AssignedOnce,
	"cli-ai-mode":       cli.AssignedNone,
	"no-cli-ai-mode":    cli.AssignedNone,
	"cli-ai-user-agent": cli.AssignedOnce,
}

// hostGlobalFlags returns the reference FlagSet describing the host's
// global flag vocabulary. Built once from config.AddFlags so it stays
// in lock-step with what the root command actually registers.
func hostGlobalFlags() *cli.FlagSet {
	hostGlobalsOnce.Do(func() {
		hostGlobalsRef = cli.NewFlagSet()
		config.AddFlags(hostGlobalsRef)
		for name, mode := range callerTransportFlags {
			hostGlobalsRef.Add(&cli.Flag{Name: name, AssignedMode: mode})
		}
	})
	return hostGlobalsRef
}

// extractHostGlobals removes recognised host global flags from args,
// applies their values to ctx, and returns the remaining args (product,
// command, engine flags, API parameters) untouched.
//
// When at least one host global is extracted, it forces the context
// into configure mode so config.LoadProfileWithContext runs
// OverwriteWithFlags and the extracted values actually override the
// profile. This makes the behaviour explicit rather than relying on
// InConfigureMode happening to default true at startup.
func extractHostGlobals(ctx *cli.Context, args []string) []string {
	ref := hostGlobalFlags()
	out := make([]string, 0, len(args))

	extracted := false
	i := 0
	for i < len(args) {
		tok := args[i]

		f, inlineVal, hasInline := matchHostFlag(ref, tok)
		if f == nil {
			out = append(out, tok)
			i++
			continue
		}

		// Consume the flag and, when it takes one, its value.
		i++
		val := inlineVal
		if !hasInline && f.AssignedMode != cli.AssignedNone {
			if i < len(args) && !strings.HasPrefix(args[i], "--") {
				val = args[i]
				i++
			}
		}
		applyHostFlag(ctx, f, val)
		extracted = true
	}

	if extracted {
		// The user explicitly passed configuration flags on the
		// command line; ensure they override the loaded profile.
		ctx.SetInConfigureMode(true)
	}
	return out
}

// matchHostFlag returns the reference flag a token denotes (long or
// shorthand), or nil when the token is not a host global. Engine-owned
// flags are treated as non-matches so they pass through.
func matchHostFlag(ref *cli.FlagSet, tok string) (f *cli.Flag, inlineVal string, hasInline bool) {
	switch {
	case strings.HasPrefix(tok, "--") && len(tok) > 2:
		name := tok[2:]
		if k, v, ok := strings.Cut(name, "="); ok {
			name, inlineVal, hasInline = k, v, true
		}
		if cand := ref.Get(name); cand != nil && !engineOwnedFlags[cand.Name] {
			return cand, inlineVal, hasInline
		}
	case len(tok) == 2 && tok[0] == '-' && tok[1] != '-':
		if cand := ref.GetByShorthand(rune(tok[1])); cand != nil && !engineOwnedFlags[cand.Name] {
			return cand, "", false
		}
	}
	return nil, "", false
}

// applyHostFlag records a host flag's value on ctx so downstream
// profile/credential resolution observes it. The flag is added to the
// context's set when absent (e.g. non-persistent flags on the KeepArgs
// path).
func applyHostFlag(ctx *cli.Context, ref *cli.Flag, val string) {
	f := ctx.Flags().Get(ref.Name)
	if f == nil {
		f = &cli.Flag{
			Name:         ref.Name,
			Shorthand:    ref.Shorthand,
			AssignedMode: ref.AssignedMode,
			Persistent:   ref.Persistent,
		}
		ctx.Flags().Add(f)
	}
	f.SetAssigned(true)
	if val != "" {
		f.SetValue(val)
	}
}
