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
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestApi_GetMethod(t *testing.T) {
	api := &Api{
		Method: "post",
	}
	method := api.GetMethod()
	assert.Equal(t, method, "POST")

	api.Method = "get"
	method = api.GetMethod()
	assert.Equal(t, method, "GET")

	api.Method = "put"
	method = api.GetMethod()
	assert.Equal(t, method, "GET")
}

func TestApi_GetProtocol(t *testing.T) {
	api := &Api{
		Protocol: "https://",
	}
	protocol := api.GetProtocol()
	assert.Equal(t, protocol, "https")

	api.Protocol = "http://"
	protocol = api.GetProtocol()
	assert.Equal(t, protocol, "http")

	api.Protocol = "HTTP"
	protocol = api.GetProtocol()
	assert.Equal(t, protocol, "http")

	api.Protocol = "HTTPS"
	protocol = api.GetProtocol()
	assert.Equal(t, protocol, "https")

	api.Protocol = "HTTP|HTTPS"
	protocol = api.GetProtocol()
	assert.Equal(t, protocol, "https")

	api.Protocol = "HTTPS|HTTP"
	protocol = api.GetProtocol()
	assert.Equal(t, protocol, "https")
}

func TestApi_FindParameter(t *testing.T) {
	api := &Api{}
	api.Parameters = []Parameter{
		{
			Name: "paramter",
		},
	}
	parameter := api.FindParameter("paramter")
	assert.Equal(t, parameter.Name, "paramter")

	api.Parameters = []Parameter{
		{
			SubParameters: []Parameter{
				{
					Name: "subparameters",
				},
			},
			Name: "paramter",
		},
	}
	parameter = api.FindParameter("paramter_test")
	assert.Nil(t, parameter)

	parameter = api.FindParameter("paramter.1.10")
	assert.Nil(t, parameter)
	parameter = api.FindParameter("paramter.1.subparameters")
	assert.NotNil(t, parameter)
	parameter = api.FindParameter("paramter.10.subparameters")
	assert.NotNil(t, parameter)
	parameter = api.FindParameter("paramter.100.subparameters")
	assert.NotNil(t, parameter)

	api.Parameters = []Parameter{
		{
			Type: "RepeatList",
			Name: "paramter",
		},
	}
	parameter = api.FindParameter("paramter.1.1")
	assert.Equal(t, parameter.Name, "paramter")

	parameter = api.FindParameter("paramter_test")
	assert.Nil(t, parameter)

	api.Parameters = nil
	parameter = api.FindParameter("paramter.1.10")
	assert.Nil(t, parameter)

	api.Parameters = []Parameter{
		{
			SubParameters: []Parameter{
				{
					Name: "subparameters",
				},
			},
			Type: "RepeatList",
			Name: "paramter",
		},
		{
			Type: "Integer",
			Name: "paramtercount",
		},
	}
	parameterCount := api.FindParameter("paramtercount")
	assert.NotNil(t, parameterCount)
	assert.Equal(t, parameterCount.Name, "paramtercount")
	parameterCount = api.FindParameter("subparameters_test")
	assert.Nil(t, parameterCount)
	parameterCount = api.FindParameter("subparameters")
	assert.Nil(t, parameterCount)
	parameter = api.FindParameter("paramter.1.10")
	assert.Nil(t, parameter)
	parameter = api.FindParameter("paramter.1.subparameters")
	assert.NotNil(t, parameter)
	parameter = api.FindParameter("paramter.10.subparameters")
	assert.NotNil(t, parameter)
	parameter = api.FindParameter("paramter.100.subparameters")
	assert.NotNil(t, parameter)
}

func TestApi_ForeachParameters(t *testing.T) {
	api := &Api{}
	f := func(s string, p Parameter) {
		p.Name = s
	}
	api.Parameters = []Parameter{
		{
			Type: "RepeatList",
			Name: "paramter",
		},
	}
	api.ForeachParameters(f)
	assert.Equal(t, api.Parameters[0].Name, "paramter")

	api.Parameters[0].Type = ""
	api.ForeachParameters(f)
	assert.Equal(t, api.Parameters[0].Name, "paramter")

	api.Parameters = []Parameter{
		{
			SubParameters: []Parameter{
				{
					Name: "subparameters",
				},
			},
			Name: "paramter",
		},
	}
	api.ForeachParameters(f)
	assert.Equal(t, api.Parameters[0].SubParameters[0].Name, "subparameters")
}

func TestApi_CheckRequiredParameters(t *testing.T) {
	api := &Api{
		Parameters: []Parameter{
			{
				Name:     "api",
				Required: true,
				Type:     "Repeat",
			},
		},
	}
	checker := func(string) bool {
		return false
	}
	err := api.CheckRequiredParameters(checker)
	assert.Contains(t, err.Error(), "required parameters not assigned")

	api.Parameters = nil
	err = api.CheckRequiredParameters(checker)
	assert.Nil(t, err)
}
func TestParameterSliceLenAndSwap(t *testing.T) {
	parameters := ParameterSlice{
		{
			Name:     "api",
			Required: true,
			Type:     "RepeatList",
		},
	}
	len := parameters.Len()
	assert.Equal(t, len, 1)

	parameters = append(parameters, Parameter{
		Name:     "api_test",
		Required: true,
		Type:     "RepeatList",
	})
	parameters.Swap(0, 1)
	assert.Equal(t, parameters[0].Name, "api_test")
	assert.Equal(t, parameters[1].Name, "api")
}

func TestParameterSlice_Less(t *testing.T) {
	parameters := ParameterSlice{
		{
			Name:     "api",
			Required: true,
			Type:     "RepeatList",
		},
		{
			Name:     "api_test",
			Required: true,
			Type:     "RepeatList",
		},
	}
	ok := parameters.Less(0, 1)
	assert.True(t, ok)

	parameters[1].Required = false
	ok = parameters.Less(0, 1)
	assert.True(t, ok)

	parameters[0].Required = false
	ok = parameters.Less(0, 1)
	assert.True(t, ok)

	parameters[1].Required = true
	ok = parameters.Less(0, 1)
	assert.False(t, ok)
}
