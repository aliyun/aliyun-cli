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
package openapi

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

// return when use unknown product
type InvalidProductError struct {
	Code    string
	library *Library
}

func (e *InvalidProductError) Error() string {
	return fmt.Sprintf("'%s' is not a valid command or product. See `aliyun help`.", strings.ToLower(e.Code))
}

func (e *InvalidProductError) GetSuggestions() []string {
	sr := cli.NewSuggester(strings.ToLower(e.Code), 2)
	for _, p := range e.library.GetProducts() {
		sr.Apply(strings.ToLower(p.Code))
	}
	return sr.GetResults()
}

// return when use unknown api
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

// return when use unknown parameter
type InvalidParameterError struct {
	Name  string
	api   *meta.Api
	flags *cli.FlagSet
}

func (e *InvalidParameterError) Error() string {
	return fmt.Sprintf("'--%s' is not a valid parameter or flag. See `aliyun help %s %s`.",
		e.Name, e.api.Product.GetLowerCode(), e.api.Name)
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
