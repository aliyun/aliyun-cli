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
package meta

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

var RepositoryTest = &Repository{}

func TestLoadRepository(t *testing.T) {
	repository := LoadRepository()
	assert.NotNil(t, repository)
	assert.Contains(t, repository.Names, "Ecs")
}

func TestGetApi(t *testing.T) {
	repository := LoadRepository()
	assert.NotNil(t, repository)
	_, ok := repository.GetApi("invalid_product", "", "")
	assert.False(t, ok)

	_, ok = repository.GetApi("ros", "", "")
	assert.False(t, ok)

	api, ok := repository.GetApi("Ecs", "2014-05-26", "DescribeRegions")
	assert.True(t, ok)
	assert.NotNil(t, api)
	assert.Equal(t, "DescribeRegions", api.Name)
}

func TestGetStyle(t *testing.T) {
	repository := LoadRepository()
	style, ok := repository.GetStyle("aegis", "2016-11-11")
	assert.True(t, ok)
	assert.Equal(t, "RPC", style)
	_, ok = repository.GetStyle("invalid_product", "2016-11-11")
	assert.False(t, ok)
}

func TestReplacePathPattern(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1",
			args: args{
				pattern: "/permissions/users/[uid]/update",
			},
			want: "/permissions/users/[0-9a-zA-Z_\\-\\.{}]+/update",
		},
		{
			name: "test2",
			args: args{
				pattern: "/permissions/users/[uid]/update/[id]",
			},
			want: "/permissions/users/[0-9a-zA-Z_\\-\\.{}]+/update/[0-9a-zA-Z_\\-\\.{}]+",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ReplacePathPattern(tt.args.pattern), "ReplacePathPattern(%v)", tt.args.pattern)
		})
	}
}

func TestRepository_GetApiByPath(t *testing.T) {
	type fields struct {
		Products []Product
		Names    []string
		index    map[string]Product
	}
	type args struct {
		productCode string
		version     string
		method      string
		path        string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Api
		want1  bool
	}{
		{
			name: "test1",
			fields: fields{
				Products: []Product{
					{
						Code:     "cs",
						ApiNames: []string{"UpdateUserPermissions"},
					},
				},
				Names: []string{"cs"},
				index: map[string]Product{
					"cs": {
						Code:     "cs",
						ApiNames: []string{"UpdateUserPermissions"},
					},
				},
			},
			args: args{
				productCode: "cs",
				version:     "2015-12-15",
				method:      "POST",
				path:        "/permissions/users/xx/update",
			},
			want1: true,
		},
		{
			name: "test2",
			fields: fields{
				Products: []Product{
					{
						Code:     "cs",
						ApiNames: []string{"UpdateUserPermissions"},
					},
				},
				Names: []string{"cs"},
				index: map[string]Product{
					"cs": {
						Code:     "cs",
						ApiNames: []string{"UpdateUserPermissions"},
					},
				},
			},
			args: args{
				productCode: "cs",
				version:     "2015-12-15",
				method:      "POST",
				path:        "/permissions/users/xx/update2",
			},
			want1: false,
		},
		{
			name: "test3",
			fields: fields{
				Products: []Product{
					{
						Code:     "cs",
						ApiNames: []string{"DeleteClusterNodes"},
					},
				},
				Names: []string{"cs"},
				index: map[string]Product{
					"cs": {
						Code:     "cs",
						ApiNames: []string{"DeleteClusterNodes"},
					},
				},
			},
			args: args{
				productCode: "cs",
				version:     "2015-12-15",
				method:      "POST",
				path:        "/clusters/clusterId/nodes",
			},
			want1: true,
		},
		{
			name: "test4",
			fields: fields{
				Products: []Product{
					{
						Code:     "cs",
						ApiNames: []string{"DeleteClusterNodes"},
					},
				},
				Names: []string{"cs"},
				index: map[string]Product{
					"cs": {
						Code:     "cs",
						ApiNames: []string{"DeleteClusterNodes"},
					},
				},
			},
			args: args{
				productCode: "cs",
				version:     "2015-12-15",
				method:      "DELETE",
				path:        "/clusters/clusterId/nodes",
			},
			want1: true,
		},
		{
			name: "test5",
			fields: fields{
				Products: []Product{
					{
						Code:     "cs",
						ApiNames: []string{"CreateTemplate"},
					},
				},
				Names: []string{"cs"},
				index: map[string]Product{
					"cs": {
						Code:     "cs",
						ApiNames: []string{"CreateTemplate"},
					},
				},
			},
			args: args{
				productCode: "cs",
				version:     "2015-12-15",
				method:      "POST",
				path:        "/templates",
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Repository{
				Products: tt.fields.Products,
				Names:    tt.fields.Names,
				index:    tt.fields.index,
			}
			_, got1 := a.GetApiByPath(tt.args.productCode, tt.args.version, tt.args.method, tt.args.path)
			assert.Equalf(t, tt.want1, got1, "GetApiByPath(%v, %v, %v, %v)", tt.args.productCode, tt.args.version, tt.args.method, tt.args.path)
		})
	}
}
