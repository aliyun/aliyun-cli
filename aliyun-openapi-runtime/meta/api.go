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

// API is the complete in-memory description of one OpenAPI operation.
// Instances are constructed by format.Format implementations from the
// on-disk representation and consumed by runtime.Executor via the
// Loader.
type API struct {
	// Identity.
	Name        string // PascalCase API name (e.g. "DescribeInstances")
	CmdName     string // kebab-case CLI name (e.g. "describe-instances")
	CmdFullName string // fully-qualified CLI path (e.g. "ecs describe-instances")

	// Ownership.
	ProductCode string // e.g. "Ecs" (as reported by upstream meta)
	Version     string // API version (e.g. "2014-05-26")

	// Wire.
	Method      string // "GET" / "POST" / ...
	URL         string // RESTful path template; empty for RPC
	Style       APIStyle
	Protocol    string            // "HTTPS" / "HTTP"
	ContentType string            // request Content-Type override, may be ""
	BodyMapping map[string]string // raw_name -> wire key, RPC body overrides

	// Content.
	Parameters  []Parameter // top-level arguments
	Endpoints   Endpoints
	Examples    []string
	Description Description

	// Lifecycle.
	Deprecated bool
}

// FindParameter returns a top-level Parameter by its Name (snake_case
// CLI key). Nested composite fields are not searched. Returns nil when
// not found.
// FindParameter looks up a top-level parameter by wire RawName first,
// then by snake_case Name. Parsed Args are keyed by RawName; Name is
// kept as a fallback for schema introspection helpers.
func (a *API) FindParameter(name string) *Parameter {
	for i := range a.Parameters {
		if a.Parameters[i].RawName == name {
			return &a.Parameters[i]
		}
	}
	for i := range a.Parameters {
		if a.Parameters[i].Name == name {
			return &a.Parameters[i]
		}
	}
	return nil
}
