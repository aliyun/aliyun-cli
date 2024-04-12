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
	"fmt"
	"strings"
)

type Api struct {
	Name        string            `json:"name"`
	Protocol    string            `json:"protocol"`
	Method      string            `json:"method"`
	PathPattern string            `json:"pathPattern"`
	Description map[string]string `json:"descriptions,omitempty"`
	Parameters  []Parameter       `json:"parameters"`
	Product     *Product          `json:"-"`
}

func (a *Api) GetMethod() string {
	method := strings.ToUpper(a.Method)
	if strings.Contains(method, "POST") {
		return "POST"
	}
	if strings.Contains(method, "GET") {
		return "GET"
	}
	return "GET"
}

func (a *Api) GetProtocol() string {
	lowered := strings.ToLower(a.Protocol)

	if strings.HasPrefix(lowered, "https") {
		return "https"
	}

	parts := strings.Split(lowered, "|")
	for _, v := range parts {
		if v == "https" {
			return "https"
		}
	}

	return "http"
}

func (a *Api) FindParameter(name string) *Parameter {
	return findParameterInner(a.Parameters, name)
}

// Foreach parameter use recursion
func (a *Api) ForeachParameters(f func(s string, p Parameter)) {
	foreachParameters(a.Parameters, "", f)
}

func foreachParameters(params []Parameter, prefix string, f func(s string, p Parameter)) {
	for _, p := range params {
		if len(p.SubParameters) > 0 {
			foreachParameters(p.SubParameters, prefix+p.Name+".1.", f)
		} else if p.Type == "RepeatList" {
			f(prefix+p.Name+".1", p)
		} else {
			f(prefix+p.Name, p)
		}
	}
}

func findParameterInner(params []Parameter, name string) *Parameter {
	for i, p := range params {
		if p.Name == name {
			return &(params[i])
		}
		if len(p.SubParameters) > 0 && strings.HasPrefix(name, p.Name+".") {
			s := name[len(p.Name):]
			// XXX.1.YYY
			if len(s) >= 4 && s[0] == '.' && strings.Count(s, ".") >= 2 {
				index := strings.Index(name[len(p.Name)+1:], ".")
				index += 2
				return findParameterInner(p.SubParameters, name[len(p.Name)+index:])
			}
			return nil
		}

		if p.Type == "RepeatList" && strings.HasPrefix(name, p.Name) {
			// XXX.1
			s := name[len(p.Name):]
			if len(s) >= 2 && s[0] == '.' {
				return &((params)[i])
			}
		}
	}
	return nil
}

func (a *Api) CheckRequiredParameters(checker func(string) bool) error {
	missing := false
	s := ""
	for _, p := range a.Parameters {
		if p.Required {
			name := p.Name
			if p.Type != "RepeatList" {
				if !checker(name) {
					missing = true
					s = s + "\n  --" + p.Name
				}
			}
		}
	}
	if missing {
		return fmt.Errorf("required parameters not assigned: " + s)
	} else {
		return nil
	}
}

type Parameter struct {
	Name          string            `json:"name"`
	Position      string            `json:"position"`
	Type          string            `json:"type"`
	Description   map[string]string `json:"description,omitempty"`
	Required      bool              `json:"required"`
	Hidden        bool              `json:"hidden"`
	Example       string            `json:"example,omitempty"`
	SubParameters []Parameter       `json:"sub_parameters,omitempty"`
}

type ParameterSlice []Parameter

func (p ParameterSlice) Len() int {
	return len(p)
}

func (p ParameterSlice) Less(i, j int) bool {
	if p[i].Required && p[j].Required {
		return p[i].Name < p[j].Name
	}

	if p[i].Required {
		return true
	}

	if p[j].Required {
		return false
	}

	return p[i].Name < p[j].Name
}

func (p ParameterSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
