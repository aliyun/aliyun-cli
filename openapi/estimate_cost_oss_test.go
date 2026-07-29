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
package openapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func newOssBridgeTestContext(t *testing.T) (*cli.Context, *cli.Command, *Commando, *bytes.Buffer) {
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{Name: "oss", Short: i18n.T("Object Storage Service", "阿里云OSS对象存储")}
	config.AddFlags(cmd.Flags())

	profile := config.NewProfile("test-oss-estimate")
	profile.Mode = "AK"
	profile.AccessKeyId = "test-ak"
	profile.AccessKeySecret = "test-secret"
	profile.RegionId = "cn-hangzhou"
	command := NewCommando(w, profile)
	command.AttachOssEstimateCost(cmd)
	ctx.EnterCommand(cmd)
	return ctx, cmd, command, w
}

// writeOssTestConfig points the config-path flag at a temp config file so
// LoadProfileWithContext resolves credentials without touching the real
// ~/.aliyun/config.json (which may not exist in CI).
func writeOssTestConfig(t *testing.T, ctx *cli.Context) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"current":"default","profiles":[{"name":"default","mode":"AK","access_key_id":"test-ak","access_key_secret":"test-secret","region_id":"cn-hangzhou"}]}`
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	f := config.ConfigurePathFlag(ctx.Flags())
	f.SetAssigned(true)
	f.SetValue(path)
}

func TestOssApiNameRegexp(t *testing.T) {
	assert.True(t, estimateCostApiNameRegexp.MatchString("CreateBucket"))
	assert.True(t, estimateCostApiNameRegexp.MatchString("PutBucketLifecycle"))
	// ossutil subcommands and misc tokens must never look like OpenAPI names.
	assert.False(t, estimateCostApiNameRegexp.MatchString("mb"))
	assert.False(t, estimateCostApiNameRegexp.MatchString("cp"))
	assert.False(t, estimateCostApiNameRegexp.MatchString("oss://bucket"))
	assert.False(t, estimateCostApiNameRegexp.MatchString("--estimate-cost"))
	assert.False(t, estimateCostApiNameRegexp.MatchString(""))
}

func TestOssBridgeLegacyBehaviorPreserved(t *testing.T) {
	ctx, cmd, command, w := newOssBridgeTestContext(t)

	// Unknown token without --estimate-cost -> invalid command, same as the
	// framework error when Run was nil.
	err := command.processOssBridgeFallthrough(ctx, cmd, []string{"CreateBucket"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CreateBucket")

	// Bare `aliyun oss` -> help text, not an error.
	w.Reset()
	err = command.processOssBridgeFallthrough(ctx, cmd, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, w.String())
}

func TestOssBridgeEstimateCostRejectsNonApiNames(t *testing.T) {
	ctx, cmd, command, _ := newOssBridgeTestContext(t)
	EstimateCostFlag(ctx.Flags()).SetAssigned(true)

	for _, args := range [][]string{{}, {"mb"}, {"oss://bucket"}} {
		err := command.processOssBridgeFallthrough(ctx, cmd, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires an OpenAPI name")
	}
}

func TestBuildOssEstimateCostParameters(t *testing.T) {
	ctx, _, _, _ := newOssBridgeTestContext(t)
	profile := config.NewProfile("p")
	profile.RegionId = "cn-shanghai"

	unknown := cli.NewFlagSet()
	f1, _ := unknown.AddByName("StorageClass")
	f1.SetAssigned(true)
	f1.SetValue("Standard")
	f2, _ := unknown.AddByName("region")
	f2.SetAssigned(true)
	f2.SetValue("cn-beijing")
	multi, _ := unknown.AddByName("Tag")
	multi.SetAssigned(true)
	multi.SetValues([]string{"a", "b"})
	ctx.SetUnknownFlags(unknown)

	parameters := buildOssEstimateCostParameters(ctx, &profile)
	assert.Equal(t, "Standard", parameters["StorageClass"])
	// --region is the bridge's region selector: consumed as RegionId fallback,
	// never forwarded as a literal `region` parameter.
	assert.Equal(t, "cn-beijing", parameters["RegionId"])
	assert.NotContains(t, parameters, "region")
	assert.Equal(t, []string{"a", "b"}, parameters["Tag"])

	// Without --region and --RegionId the profile region fills in.
	ctx.SetUnknownFlags(cli.NewFlagSet())
	parameters = buildOssEstimateCostParameters(ctx, &profile)
	assert.Equal(t, "cn-shanghai", parameters["RegionId"])

	// Explicit --RegionId wins over both.
	unknown2 := cli.NewFlagSet()
	f3, _ := unknown2.AddByName("RegionId")
	f3.SetAssigned(true)
	f3.SetValue("cn-hongkong")
	ctx.SetUnknownFlags(unknown2)
	parameters = buildOssEstimateCostParameters(ctx, &profile)
	assert.Equal(t, "cn-hongkong", parameters["RegionId"])
}

func TestOssBridgeEstimateCostFlow(t *testing.T) {
	// Same sentinel-endpoint contract as the RPC and openapi-path tests: the
	// flow must reach the estimate-cost client (DNS failure on the sentinel
	// host proves interception) without touching ossutil or any OSS endpoint.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")

	ctx, cmd, command, _ := newOssBridgeTestContext(t)
	writeOssTestConfig(t, ctx)
	EstimateCostFlag(ctx.Flags()).SetAssigned(true)

	unknown := cli.NewFlagSet()
	f, _ := unknown.AddByName("StorageClass")
	f.SetAssigned(true)
	f.SetValue("Standard")
	ctx.SetUnknownFlags(unknown)

	err := command.processOssBridgeFallthrough(ctx, cmd, []string{"CreateBucket"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")
}

func TestEstimateOssCostHermetic(t *testing.T) {
	// Direct entry-point coverage (the ossutil passthrough calls this from
	// another package): profile from a temp config, quote routed at a fake
	// endpoint — the dial error proves parameters/pricing-context handling ran.
	t.Setenv(estimateCostEndpointEnv, "estimate-cost.test.invalid")
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, w)
	cmd := &cli.Command{Name: "ossutil"}
	AddFlags(cmd.Flags())
	config.AddFlags(cmd.Flags())
	ctx.EnterCommand(cmd)

	configPath := filepath.Join(t.TempDir(), "config.json")
	assert.NoError(t, os.WriteFile(configPath, []byte(`{"current":"t","profiles":[{"name":"t","mode":"AK","access_key_id":"ak","access_key_secret":"sk","region_id":"cn-hangzhou"}]}`), 0o600))
	pathFlag := config.ConfigurePathFlag(ctx.Flags())
	pathFlag.SetAssigned(true)
	pathFlag.SetValue(configPath)

	err := EstimateOssCost(ctx, "PutBucket",
		map[string]interface{}{"StorageClass": "Standard"},
		map[string]interface{}{"EstimatedStorageGB": 100})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")

	// Nil parameters must not panic; profile region backfills RegionId.
	err = EstimateOssCost(ctx, "PutBucket", nil, nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "estimate-cost.test.invalid")

	// Unconfigured profile fails with the actionable configure tip.
	badPath := filepath.Join(t.TempDir(), "empty.json")
	assert.NoError(t, os.WriteFile(badPath, []byte(`{"current":"t","profiles":[{"name":"t"}]}`), 0o600))
	pathFlag.SetValue(badPath)
	err = EstimateOssCost(ctx, "PutBucket", nil, nil)
	assert.NotNil(t, err)
}
