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

package loader

import (
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestValidateAPIParameterOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []string
		wantErr string
	}{
		{name: "valid", options: []string{"--region-id"}},
		{name: "missing", wantErr: "exactly one CLI option, got 0"},
		{name: "multiple", options: []string{"--region-id", "--RegionId"}, wantErr: "exactly one CLI option, got 2"},
		{name: "empty", options: []string{""}, wantErr: `invalid CLI option ""`},
		{name: "not long form", options: []string{"region-id"}, wantErr: `invalid CLI option "region-id"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &meta.API{Parameters: []meta.Parameter{{
				Name:    "region_id",
				Options: tt.options,
				Fields: []meta.Parameter{{
					Name: "nested_field",
				}},
			}}}
			err := validateAPIParameterOptions(api)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateAPIParameterOptions: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
