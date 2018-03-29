package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"strings"
)

type InvalidProductError struct {
	Code    string
	library *meta.Library
}

func (e *InvalidProductError) Error() string {
	return fmt.Sprintf("'%s' is not a valid command or product. See `aliyun help`.", strings.ToLower(e.Code))
}

func (e *InvalidProductError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.library.Products {
		sr.Apply(strings.ToLower(p.Code))
	}
	return sr.GetResults()
}

type InvalidApiError struct {
	Name    string
	product *meta.Product
}

func (e *InvalidApiError) Error() string {
	return fmt.Sprintf("'%s' is not a valid api. See `aliyun help %s`.", e.Name, e.product.GetLowerCode())
}

func (e *InvalidApiError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, s := range e.product.ApiNames {
		sr.Apply(s)
	}
	return sr.GetResults()
}

type InvalidParameterError struct {
	Name      string
	Shorthand string
	api       *meta.Api
	flags     *cli.FlagSet
}

func (e *InvalidParameterError) Error() string {
	var param string
	if e.Name != "" {
		param = "--" + e.Name
	} else {
		param = "-" + e.Shorthand
	}
	return fmt.Sprintf("'%s' is not a valid parameter or flag. See `aliyun help %s %s`.",
		param, e.api.Product.GetLowerCode(), e.api.Name)
}

func (e *InvalidParameterError) GetSuggestions() []string {
	sr := cli.NewSuggester(e.Name, 2)
	for _, p := range e.api.Parameters {
		sr.Apply(p.Name)
	}
	for _, f := range e.flags.Flags() {
		sr.Apply(f.Name)
	}
	return sr.GetResults()
}
