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

package runtimehost

import (
	"bytes"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
	"github.com/stretchr/testify/require"
)

type safetyPluginSource struct{ source.Source }

func (s safetyPluginSource) LoadProduct(code string) (*meta.Product, *source.Provenance, error) {
	p, provenance, err := s.Source.LoadProduct(code)
	if provenance != nil {
		provenance.Kind = source.KindUser
	}
	return p, provenance, err
}

func TestSafetyAPICommandMetadataBoundaries(t *testing.T) {
	old := Engine()
	t.Cleanup(func() { engineInst = old })
	fsys := fstest.MapFS{
		"metadatas/products.json":         {Data: []byte(`{"products":[{"code":"demo","plugin_default_version":"v1","versions":["v1","v2","bad","missing"]}]}`)},
		"canonical/demo/v1/version.json":  {Data: []byte(`{"version":"v1","apis":{"DeleteThing":{"cmd_name":"delete-thing"}}}`)},
		"canonical/demo/v2/version.json":  {Data: []byte(`{"version":"v2","apis":{"DeleteThing":{"cmd_name":"remove-thing"}}}`)},
		"canonical/demo/bad/version.json": {Data: []byte(`invalid`)},
	}
	for _, plugin := range []bool{false, true} {
		var src source.Source = source.NewBaselineSource(fsys, "test")
		if plugin {
			src = safetyPluginSource{src}
		}
		ldr := loader.New(src)
		engineInst = engine.NewEngine(func() (loader.Loader, error) { return ldr, nil }, nil)
		for _, tc := range []struct {
			command string
			args    []string
			matched bool
		}{
			{"delete-thing", nil, true},
			{"remove-thing", []string{"--api-version", "v2"}, true},
			{"remove-thing", []string{"--api-version=v2"}, true},
			{"remove-thing", nil, false},
			{"delete-thing", []string{"--api-version", "v2"}, false},
			{"delete-thing", []string{"--api-version", "unknown"}, false},
			{"delete-thing", []string{"--api-version", "bad"}, false},
			{"delete-thing", []string{"--api-version", "missing"}, false},
			{"unknown", nil, false},
		} {
			p := &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "demo:DeleteThing", Action: safety.ActionDeny}}}
			cmd := safetyAPICommand(safety.CommandInfo{Product: "demo", ApiOrMethod: tc.command}, tc.args)
			require.Equal(t, tc.matched && !plugin, p.Check(cmd).Matched, "plugin=%v, case=%+v", plugin, tc)
			p.Rules[0].Pattern = "demo:" + tc.command
			require.True(t, p.Check(cmd).Matched, "syntactic fallback must survive")
		}
		t.Setenv("ALIBABA_CLOUD_DEMO_API_VERSION", "v2")
		p := &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "demo:DeleteThing", Action: safety.ActionDeny}}}
		cmd := safetyAPICommand(safety.CommandInfo{Product: "demo", ApiOrMethod: "remove-thing"}, nil)
		require.Equal(t, !plugin, p.Check(cmd).Matched)
		t.Setenv("ALIBABA_CLOUD_DEMO_API_VERSION", "")
	}
	engineInst = engine.NewEngine(func() (loader.Loader, error) { return nil, errors.New("missing metadata") }, nil)
	cmd := safetyAPICommand(safety.CommandInfo{Product: "demo", ApiOrMethod: "delete-thing"}, nil)
	p := &safety.Policy{Enabled: true, Rules: []safety.Rule{{Pattern: "demo:delete-thing", Action: safety.ActionDeny}}}
	require.True(t, p.Check(cmd).Matched)
}

func TestDispatchSafetyIdentityConfirmAndDeny(t *testing.T) {
	old := Engine()
	engineInst = baselineEngine(t)
	t.Cleanup(func() { engineInst = old })
	for _, action := range []safety.Action{safety.ActionConfirm, safety.ActionDeny} {
		for _, skip := range []bool{false, true} {
			t.Run(string(action)+"/skip="+map[bool]string{false: "false", true: "true"}[skip], func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv(safety.EnvSafetyPolicyEnabled, "true")
				t.Setenv(safety.EnvSafetyPolicyRules, "ecs:*=allow,ecs:DeleteInstance="+string(action))
				t.Setenv("ALIBABA_CLOUD_SAFETY_SKIP_CONFIRM", map[bool]string{false: "false", true: "true"}[skip])
				ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
				ctx.Flags().Add(config.NewConfigurePathFlag())
				config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
				config.ConfigurePathFlag(ctx.Flags()).SetValue(dir + "/config.json")
				called := false
				previous := engineDispatch
				engineDispatch = func(engine.Request) error { called = true; return nil }
				t.Cleanup(func() { engineDispatch = previous })
				err := Dispatch(ctx, []string{"ecs", "delete-instance"})
				if action == safety.ActionConfirm && skip {
					require.NoError(t, err)
					require.True(t, called)
				} else {
					require.Error(t, err)
					require.False(t, called)
				}
			})
		}
	}
}

func TestSafetyDisabledDoesNotLoadMetadata(t *testing.T) {
	old := Engine()
	t.Cleanup(func() { engineInst = old })
	calls := 0
	engineInst = engine.NewEngine(func() (loader.Loader, error) { calls++; return nil, errors.New("must not load") }, nil)
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.Flags().Add(config.NewConfigurePathFlag())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(t.TempDir() + "/config.json")
	t.Setenv(safety.EnvSafetyPolicyEnabled, "false")
	t.Setenv(safety.EnvSafetyPolicyRules, "ecs:DeleteInstance=deny")
	require.NoError(t, checkSafetyPolicy(ctx, []string{"ecs", "delete-instance"}))
	t.Setenv(safety.EnvSafetyPolicyEnabled, "true")
	t.Setenv(safety.EnvSafetyPolicyRules, "")
	require.NoError(t, checkSafetyPolicy(ctx, []string{"ecs", "delete-instance"}))
	require.Zero(t, calls)
}
