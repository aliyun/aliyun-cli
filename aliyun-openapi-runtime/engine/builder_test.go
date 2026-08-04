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

package engine

import (
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestBuildExecContextRawBodyPrecedesBodyFile(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{
		Body:        `{"from":"body"}`,
		BodySet:     true,
		BodyFile:    "/file/that/must/not/be/read",
		BodyFileSet: true,
		DryRun:      true,
	}}
	ec, err := buildExecContext(Request{}, &meta.API{}, res)
	if err != nil {
		t.Fatalf("buildExecContext: %v", err)
	}
	if ec.RawBody != res.Reserved.Body {
		t.Fatalf("raw body = %#v, want %#v", ec.RawBody, res.Reserved.Body)
	}
}

func TestBuildExecContextPreservesExplicitEmptyRawBody(t *testing.T) {
	res := &argparser.Result{Reserved: argparser.Reserved{BodySet: true, DryRun: true}}
	ec, err := buildExecContext(Request{}, &meta.API{}, res)
	if err != nil {
		t.Fatalf("buildExecContext: %v", err)
	}
	if ec.RawBody == nil || ec.RawBody != "" {
		t.Fatalf("raw body = %#v, want explicit empty string", ec.RawBody)
	}
}
