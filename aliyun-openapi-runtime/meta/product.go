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

// Product is the top-level unit of publication: one Alibaba Cloud
// service (e.g. "ecs") that owns one or more API versions.
type Product struct {
	Code           string // short code, e.g. "ecs"
	Name           Description
	Versions       []string
	DefaultVersion string
	MinCliVersion  string

	// Description is user-facing help text shown in `aliyun --help`
	// output.
	Description Description

	// Endpoints are the product-level region -> host mappings (the
	// canonical layout carries endpoints per product, not per API).
	// The Source injects these into each meta.API's Endpoints on load.
	Endpoints Endpoints
}

// HasVersion reports whether v is listed in Versions.
func (p *Product) HasVersion(v string) bool {
	for _, x := range p.Versions {
		if x == v {
			return true
		}
	}
	return false
}

// ============================================================================
// Lightweight index
// ============================================================================

// APIIndexEntry is the minimum information required to render a
// command stub without loading the full API meta. Multiple entries
// share a Version and ProductCode, which is why those fields are
// pushed into APIIndex (the container) rather than repeated here.
type APIIndexEntry struct {
	APIName     string // PascalCase, matches API.Name and the file basename
	CmdName     string
	CmdFullName string
	Description Description
	Deprecated  bool
}

// APIIndex groups the lightweight entries of one (product, version)
// pair. The Loader reads it while resolving a command; the full per-API
// meta is fetched lazily on first use.
type APIIndex struct {
	ProductCode string
	Version     string
	Entries     map[string]APIIndexEntry // key = APIName

	// cmdIndex is the reverse map cmd-name -> APIName, built once when
	// the index is decoded so command resolution is an O(1) lookup
	// instead of a scan over Entries. Access it through ResolveCmd.
	cmdIndex map[string]string
}

// Names returns every APIName in the index in an unspecified order.
// Callers that need deterministic iteration should sort the result.
func (i *APIIndex) Names() []string {
	if i == nil || len(i.Entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(i.Entries))
	for k := range i.Entries {
		out = append(out, k)
	}
	return out
}

// BuildCmdIndex (re)builds the cmd-name -> APIName reverse map from
// Entries. It is idempotent and safe to call once after the index is
// fully populated. Later entries win on the (rare) cmd-name collision,
// matching the last-writer semantics of a plain map build.
func (i *APIIndex) BuildCmdIndex() {
	if i == nil {
		return
	}
	m := make(map[string]string, len(i.Entries))
	for apiName, e := range i.Entries {
		if e.CmdName != "" {
			m[e.CmdName] = apiName
		}
	}
	i.cmdIndex = m
}

// ResolveCmd returns the APIName bound to a kebab command name, or ""
// when unknown. When the reverse map has not been built (BuildCmdIndex
// was not called) it falls back to a read-only scan, so it is always
// correct and never mutates a shared/cached index.
func (i *APIIndex) ResolveCmd(cmd string) string {
	if i == nil || cmd == "" {
		return ""
	}
	if i.cmdIndex != nil {
		return i.cmdIndex[cmd]
	}
	for apiName, e := range i.Entries {
		if e.CmdName == cmd {
			return apiName
		}
	}
	return ""
}

// ============================================================================
// Command routing
// ============================================================================

// APIRef is the canonical three-tuple that identifies one API in the
// runtime: <product code, API version, PascalCase API name>. It is
// returned by the Loader's command router so higher layers can hand
// it to GetAPI without repeatedly re-parsing the user's argv.
//
// APIRef is deliberately a value type (small, comparable) so it can
// serve as a map key or be safely copied across goroutine boundaries.
type APIRef struct {
	Product string // kebab, e.g. "ecs"
	Version string // e.g. "2014-05-26"
	Name    string // PascalCase APIName, e.g. "DescribeInstances"
}

// String returns "<product>/<version>/<APIName>" for logs and errors.
func (r APIRef) String() string {
	return r.Product + "/" + r.Version + "/" + r.Name
}

// IsZero reports whether r is the empty reference.
func (r APIRef) IsZero() bool {
	return r.Product == "" && r.Version == "" && r.Name == ""
}
