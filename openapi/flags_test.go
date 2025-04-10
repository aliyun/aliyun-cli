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
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func TestAFlags(t *testing.T) {
	i18n.SetLanguage("en")
	flagset := cli.NewFlagSet()
	AddFlags(flagset)
	secureflag := SecureFlag(flagset)
	assert.Equal(t, "secure", secureflag.Name)
	assert.Equal(t, "use `--secure` to force https", secureflag.Short.Text())

	forceflag := ForceFlag(flagset)
	assert.Equal(t, "force", forceflag.Name)
	assert.Equal(t, "use `--force` to skip api and parameters check", forceflag.Short.Text())

	endpointflag := EndpointFlag(flagset)
	assert.Equal(t, "endpoint", endpointflag.Name)
	assert.Equal(t, "use `--endpoint <endpoint>` to assign endpoint", endpointflag.Short.Text())

	versionflag := VersionFlag(flagset)
	assert.Equal(t, "version", versionflag.Name)
	assert.Equal(t, "use `--version <YYYY-MM-DD>` to assign product api version", versionflag.Short.Text())

	headerflag := HeaderFlag(flagset)
	assert.Equal(t, "header", headerflag.Name)
	assert.Equal(t, "use `--header X-foo=bar` to add custom HTTP header, repeatable", headerflag.Short.Text())

	bodyflag := BodyFlag(flagset)
	assert.Equal(t, "body", bodyflag.Name)
	assert.Equal(t, "use `--body $(cat foo.json)` to assign http body in RESTful call", bodyflag.Short.Text())

	bodyfileflag := BodyFileFlag(flagset)
	assert.Equal(t, "body-file", bodyfileflag.Name)
	assert.Equal(t, "assign http body in Restful call with local file", bodyfileflag.Short.Text())

	acceptflag := AcceptFlag(flagset)
	assert.Equal(t, "accept", acceptflag.Name)
	assert.Equal(t, "add `--accept {json|xml}` to add Accept header", acceptflag.Short.Text())

	roaflag := RoaFlag(flagset)
	assert.Equal(t, "roa", roaflag.Name)
	assert.Equal(t, "use `--roa {GET|PUT|POST|DELETE}` to assign restful call.[DEPRECATED]", roaflag.Short.Text())

	dryrunflag := DryRunFlag(flagset)
	assert.Equal(t, "dryrun", dryrunflag.Name)
	assert.Equal(t, "add `--dryrun` to validate and print request without running.", dryrunflag.Short.Text())

	quietflag := QuietFlag(flagset)
	assert.Equal(t, "quiet", quietflag.Name)
	assert.Equal(t, "add `--quiet` to hide normal output", quietflag.Short.Text())

	outputflag := OutputFlag(flagset)
	assert.Equal(t, "output", outputflag.Name)
	assert.Equal(t, "use `--output cols=Field1,Field2 [rows=jmesPath]` to print output as table", outputflag.Short.Text())

	methodflag := MethodFlag(flagset)
	assert.Equal(t, "method", methodflag.Name)
	assert.Equal(t, "add `--method {GET|POST}` to assign rpc call method.", methodflag.Short.Text())
}
