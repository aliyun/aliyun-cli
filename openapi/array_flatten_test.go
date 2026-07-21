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

	"github.com/stretchr/testify/assert"
)

func TestExpandJSONArrayParameter_ObjectArray(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Servers", `[{"ServerId":"i-xxx","ServerType":"Ecs","Port":8081,"Weight":100}]`)
	assert.True(t, ok)
	assert.Equal(t, "i-xxx", flat["Servers.1.ServerId"])
	assert.Equal(t, "Ecs", flat["Servers.1.ServerType"])
	assert.Equal(t, "8081", flat["Servers.1.Port"])
	assert.Equal(t, "100", flat["Servers.1.Weight"])
}

func TestExpandJSONArrayParameter_MultipleItems(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Servers", `[{"ServerId":"i-1","Port":80},{"ServerId":"i-2","Port":443}]`)
	assert.True(t, ok)
	assert.Equal(t, "i-1", flat["Servers.1.ServerId"])
	assert.Equal(t, "80", flat["Servers.1.Port"])
	assert.Equal(t, "i-2", flat["Servers.2.ServerId"])
	assert.Equal(t, "443", flat["Servers.2.Port"])
}

func TestExpandJSONArrayParameter_PrimitiveArray(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Ids", `["a","b"]`)
	assert.True(t, ok)
	assert.Equal(t, "a", flat["Ids.1"])
	assert.Equal(t, "b", flat["Ids.2"])
}

func TestExpandJSONArrayParameter_SingleObject(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Server", `{"ServerId":"i-xxx","Port":8081}`)
	assert.True(t, ok)
	assert.Equal(t, "i-xxx", flat["Server.ServerId"])
	assert.Equal(t, "8081", flat["Server.Port"])
}

func TestExpandJSONArrayParameter_NestedArray(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Rule", `[{"Actions":[{"Type":"Forward","Order":1}]}]`)
	assert.True(t, ok)
	assert.Equal(t, "Forward", flat["Rule.1.Actions.1.Type"])
	assert.Equal(t, "1", flat["Rule.1.Actions.1.Order"])
}

func TestExpandJSONArrayParameter_BoolAndNull(t *testing.T) {
	flat, ok := expandJSONArrayParameter("Items", `[{"Enabled":true,"Note":null}]`)
	assert.True(t, ok)
	assert.Equal(t, "true", flat["Items.1.Enabled"])
	_, hasNote := flat["Items.1.Note"]
	assert.False(t, hasNote)
}

func TestExpandJSONArrayParameter_RejectsNonJSON(t *testing.T) {
	_, ok := expandJSONArrayParameter("Servers", "not-json")
	assert.False(t, ok)

	_, ok = expandJSONArrayParameter("Servers", "")
	assert.False(t, ok)

	_, ok = expandJSONArrayParameter("Servers", "   ")
	assert.False(t, ok)

	_, ok = expandJSONArrayParameter("Servers", `{broken`)
	assert.False(t, ok)

	_, ok = expandJSONArrayParameter("Servers", `[]`)
	assert.False(t, ok)
}

func TestFlattenRPCValue_DefaultKind(t *testing.T) {
	out := make(map[string]string)
	flattenRPCValue("N", float64(1.5), out)
	assert.Equal(t, "1.5", out["N"])
}
