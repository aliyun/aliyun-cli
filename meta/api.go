/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package meta

import (
	"strings"
	"fmt"
)

type Api struct {
	Name        string            `yaml:"name"`
	Method      string            `yaml:"method"`
	Protocol    string            `yaml:"protocol"`
	Description map[string]string `yaml:"description"`
	DocumentId  string            `yaml:"document_id"`
	Parameters  []Parameter       `yaml:"parameters"`
	Product     *Product 		  `yaml:"-"`
}

type Parameter struct {
	Name          string            `yaml:"name"`
	Type          string            `yaml:"type"`
	Description   map[string]string `yaml:"description"`
	Required      bool              `yaml:"required"`
	Hidden        bool              `yaml:"hidden"`
	Example       string            `yaml:"example"`
	SubParameters []Parameter       `yaml:""`
}

//
// TODO: make grace implementation
func (a *Api) HasParameter(name string) bool {
	if strings.Contains(name, ".") {
		// TODO: process sub parameter
		return true
	}
	for _, p := range a.Parameters {
		if name == p.Name {
			return true
		}
	}
	return false
}

func (a *Api) CheckRequiredParameters(checker func(string) bool) error {
	for _, p := range a.Parameters {
		if p.Required && !checker(p.Name) {
			return fmt.Errorf("required parameter %s not assigned", p.Name)
		}
	}
	return nil
}
