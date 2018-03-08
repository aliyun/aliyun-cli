/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package meta

import (
	"strings"
	"fmt"
)

type Api struct {
	Name        string            `json:"name"`
	Protocol    string            `json:"protocol"`
	Method      string            `json:"method"`
	Description map[string]string `json:"descriptions,omitempty"`
	Parameters  []Parameter       `json:"parameters"`
	Product     *Product 		  `json:"-"`
}

type Parameter struct {
	Name          string            `json:"name"`
	Position	  string 			`json:"position"`
	Type          string            `json:"type"`
	Description   map[string]string `json:"description,omitempty"`
	Required      bool              `json:"required"`
	Hidden        bool              `json:"hidden"`
	Example       string            `json:"example,omitempty"`
	SubParameters []Parameter       `json:"sub_parameters,omitempty"`
}

func (a *Api) FindParameter(name string) *Parameter {
	return findParameterInner(&a.Parameters, name)
}

func findParameterInner(params *[]Parameter, name string) *Parameter {
	for i, p := range *params {
		if p.Name == name {
			return &((*params)[i])
		}
		if len(p.SubParameters) > 0 && strings.HasPrefix(name, p.Name){
			s := name[len(p.Name):]
			if len(s) >= 4 && s[0] == '.' && s[2] == '.' {
				return findParameterInner(&p.SubParameters, name[len(p.Name) + 3:])
			} else {
				return nil
			}
		}
	}
	return nil
}

func (a *Api) GetDocumentLink() string {
	return fmt.Sprintf("https://help.aliyun.com/api/%s/%s.html", strings.ToLower(a.Product.Code), a.Name)
}

//
// support Name or Name.1.xxx.2.xxx
func (a *Api) HasParameter(name string) bool {
	return a.FindParameter(name) != nil
}

func (a *Api) CheckRequiredParameters(checker func(string) bool) error {
	for _, p := range a.Parameters {
		if p.Required && !checker(p.Name) {
			return fmt.Errorf("required parameter %s not assigned", p.Name)
		}
	}
	return nil
}
