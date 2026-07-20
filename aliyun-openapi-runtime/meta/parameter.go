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

// Parameter is the recursive descriptor of one CLI argument and its
// mapping to a wire-level API field.
//
// Composite kinds:
//   - Object: fields listed in Fields; each field is a full Parameter.
//   - Array:  element shape in ItemType.
//   - Map:    key type is always string; value shape in ValueType.
//
// Only the fields relevant to the parameter's Type are populated;
// consumers MUST tolerate zero values for the others.
type Parameter struct {
	// Identity.
	Name    string // snake_case CLI-facing key (e.g. "region_id")
	RawName string // wire-side field name (e.g. "RegionId")

	// Semantics.
	Type     DataType
	Position Position // top-level only; inner fields ignore this
	Required bool
	Default  any      // parsed default, wire form
	Enum     []string // legal values for scalar types

	// UI.
	Options     []string    // command-line aliases, e.g. ["--region-id"]
	Description Description // help_zh / help_en

	// Wire style hints (mostly meaningful for arrays / body composition).
	//
	//   repeatList  -> --tag k=v --tag k=v
	//   bracketList -> --tag.1=k --tag.2=v
	//   json        -> --tag '[{"k":"v"}]'
	ParamStyle string

	// Composite descriptors.
	Fields    []Parameter // Type == TypeObject
	ItemType  *Parameter  // Type == TypeArray
	ValueType *Parameter  // Type == TypeMap
}

// IsComposite reports whether the parameter has structural children.
func (p Parameter) IsComposite() bool {
	switch p.Type {
	case TypeObject, TypeArray, TypeMap:
		return true
	default:
		return false
	}
}

// WalkFields visits every descendant Parameter of an object/array/map
// tree in depth-first order, invoking fn for each node. If fn returns
// false the walk stops for that subtree.
//
// The root parameter itself is NOT visited; use `fn(p) && p.WalkFields(fn)`
// if you also need the root.
func (p Parameter) WalkFields(fn func(Parameter) bool) {
	switch p.Type {
	case TypeObject:
		for _, f := range p.Fields {
			if !fn(f) {
				continue
			}
			f.WalkFields(fn)
		}
	case TypeArray:
		if p.ItemType != nil {
			if fn(*p.ItemType) {
				p.ItemType.WalkFields(fn)
			}
		}
	case TypeMap:
		if p.ValueType != nil {
			if fn(*p.ValueType) {
				p.ValueType.WalkFields(fn)
			}
		}
	}
}
