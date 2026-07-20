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
	"bytes"
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestExtractHostGlobals(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))

	in := []string{
		"ecs", "describe-instances",
		"--profile", "prod", // host global (value)
		"--read-timeout", "30", // host global (value)
		"--region", "cn-beijing", // ENGINE-owned: must pass through
		"--endpoint", "ecs.example.com", // ENGINE-owned: pass through
		"--instance-id", "i-123", // API param: pass through
		"--cli-dry-run", // engine reserved: pass through
	}
	got := extractHostGlobals(ctx, in)

	want := []string{
		"ecs", "describe-instances",
		"--region", "cn-beijing",
		"--endpoint", "ecs.example.com",
		"--instance-id", "i-123",
		"--cli-dry-run",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining args = %v\nwant %v", got, want)
	}

	// Host globals were applied to ctx.
	if v, _ := ctx.Flags().GetValue("profile"); v != "prod" {
		t.Errorf("profile not applied to ctx: %q", v)
	}
	if v, _ := ctx.Flags().GetValue("read-timeout"); v != "30" {
		t.Errorf("read-timeout not applied to ctx: %q", v)
	}
	// Engine-owned flags must NOT have been consumed into ctx as host
	// globals (they stayed in the args for the engine).
}

func TestExtractHostGlobalsInlineAndShorthand(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	// -p is the profile shorthand; --mode=AK inline value.
	in := []string{"ecs", "describe-regions", "-p", "staging", "--mode=AK", "--cli-dry-run-json"}
	got := extractHostGlobals(ctx, in)
	want := []string{"ecs", "describe-regions", "--cli-dry-run-json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
	if v, _ := ctx.Flags().GetValue("profile"); v != "staging" {
		t.Errorf("shorthand profile not applied: %q", v)
	}
	if v, _ := ctx.Flags().GetValue("mode"); v != "AK" {
		t.Errorf("inline mode not applied: %q", v)
	}
}

func TestExtractHostGlobalsNoneLeavesArgsUntouched(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	in := []string{"ecs", "run-instances", "--image-id", "img-1", "--region", "cn-x", "--cli-dry-run"}
	got := extractHostGlobals(ctx, in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("args should be untouched when no host globals; got %v", got)
	}
	// No host global extracted -> must not force configure mode.
	if ctx.InConfigureMode() {
		t.Fatal("configure mode should not be forced when no host global is present")
	}
}

// TestExtractHostGlobalsForcesConfigureMode locks the hardening: once a
// host global is extracted, the context is put in configure mode so
// OverwriteWithFlags applies the extracted values (no longer relying on
// InConfigureMode defaulting true at startup).
func TestExtractHostGlobalsForcesConfigureMode(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	if ctx.InConfigureMode() {
		t.Fatal("precondition: fresh ctx should not be in configure mode")
	}
	_ = extractHostGlobals(ctx, []string{"ecs", "describe-regions", "--mode", "AK"})
	if !ctx.InConfigureMode() {
		t.Fatal("configure mode must be forced after extracting a host global")
	}
}

func TestExtractHostGlobalsTransportFlags(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	in := []string{
		"ecs", "describe-regions",
		"--user-agent", "tool/1",
		"--cli-ai-mode",
		"--skip-secure-verify",
		"--cli-dry-run",
	}
	got := extractHostGlobals(ctx, in)
	want := []string{"ecs", "describe-regions", "--cli-dry-run"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
	if v, _ := ctx.Flags().GetValue("user-agent"); v != "tool/1" {
		t.Errorf("user-agent = %q", v)
	}
	if f := ctx.Flags().Get("cli-ai-mode"); f == nil || !f.IsAssigned() {
		t.Error("cli-ai-mode not assigned")
	}
	if f := ctx.Flags().Get("skip-secure-verify"); f == nil || !f.IsAssigned() {
		t.Error("skip-secure-verify not assigned")
	}
}
