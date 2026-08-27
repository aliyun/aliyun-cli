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

package argparser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// paramIndex resolves a user-typed flag name to its Parameter via Parameter.Options only (minus the "--").
// RawName is only the Args map key after a flag resolves.
// A parameter with empty Options is unreachable from argv.
type paramIndex struct {
	byName map[string]*meta.Parameter
	params []meta.Parameter
}

func newParamIndex(params []meta.Parameter) *paramIndex {
	pi := &paramIndex{
		byName: make(map[string]*meta.Parameter, len(params)),
		params: params,
	}
	for i := range params {
		p := &pi.params[i]
		for _, opt := range p.Options {
			pi.register(strings.TrimPrefix(opt, "--"), p)
		}
	}
	return pi
}

func (pi *paramIndex) register(alias string, p *meta.Parameter) {
	if alias == "" {
		return
	}
	if _, exists := pi.byName[alias]; exists {
		return
	}
	pi.byName[alias] = p
}

func (pi *paramIndex) lookup(name string) *meta.Parameter {
	if p, ok := pi.byName[name]; ok {
		return p
	}
	return nil
}

// optionNames returns the sorted, user-facing option names (without "--") for error suggestions — exactly the registered Options aliases.
func (pi *paramIndex) optionNames() []string {
	seen := map[string]struct{}{}
	for i := range pi.params {
		for _, opt := range pi.params[i].Options {
			name := strings.TrimPrefix(opt, "--")
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// UnknownFlagError is returned when a flag matches neither a reserved
// name nor any schema parameter. It carries the known option names so
// the CLI layer can render a "did you mean" hint.
type UnknownFlagError struct {
	Flag  string
	Known []string
}

func (e *UnknownFlagError) Error() string {
	return fmt.Sprintf("unknown flag --%s", e.Flag)
}

func (*UnknownFlagError) AIRecoveryEligible() {}
