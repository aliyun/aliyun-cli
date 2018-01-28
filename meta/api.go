package meta

import (
	"strings"
	"fmt"
)

type Api struct {
	Name string				`yaml:"name"`
	Parameters []Parameter	`yaml:"parameters"`
}

type Parameter struct {
	Name string		`yaml:"name"`
	Type string 	`yaml:"type"`
	Required bool	`yaml:"required"`
	SubParameters []Parameter	`yaml:""`
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

func (a *Api) CheckRequiredParameters(checker func (string) bool) error {
	for _, p := range a.Parameters {
		if !checker(p.Name) {
			return fmt.Errorf("required parameter %s not assigned", p.Name)
		}
	}
	return nil
}