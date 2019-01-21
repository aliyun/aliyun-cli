package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestInvalidProductError_Error(t *testing.T) {
	err := &InvalidProductError{
		Code: "ecs",
	}
	str := err.Error()
	assert.Equal(t, "'ecs' is not a valid command or product. See `aliyun help`.", str)
}

func TestInvalidProductError_GetSuggestions(t *testing.T) {
	err := &InvalidProductError{
		Code: "ecs",
		library: &Library{
			builtinRepo: &meta.Repository{
				Products: []meta.Product{
					{
						Code: "ecs",
					},
				},
			},
		},
	}
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "ecs")
}

func TestInvalidApiError_Error(t *testing.T) {
	err := &InvalidApiError{
		Name: "describeregion",
		product: &meta.Product{
			Code: "ecs",
		},
	}
	str := err.Error()
	assert.Equal(t, "'describeregion' is not a valid api. See `aliyun help ecs`.", str)
}

func TestInvalidApiError_GetSuggestions(t *testing.T) {
	err := &InvalidApiError{
		Name: "describeregion",
		product: &meta.Product{
			Code:     "ecs",
			ApiNames: []string{"describeregion"},
		},
	}
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "describeregion")
}

func TestInvalidParameterError_Error(t *testing.T) {
	err := &InvalidParameterError{
		Name: "ak",
		api: &meta.Api{
			Name: "describeregion",
			Product: &meta.Product{
				Code: "ecs",
			},
		},
	}
	str := err.Error()
	assert.Equal(t, "'--ak' is not a valid parameter or flag. See `aliyun help ecs describeregion`.", str)
}

func TestInvalidParameterError_GetSuggestions(t *testing.T) {
	err := &InvalidParameterError{
		Name: "secure",
		api: &meta.Api{
			Name: "describeregion",
			Parameters: []meta.Parameter{
				{
					Name: "test",
				},
			},
		},
		flags: cli.NewFlagSet(),
	}
	AddFlags(err.flags)
	err.GetSuggestions()
	arrstr := err.GetSuggestions()
	str := strings.Join(arrstr, ",")
	assert.Contains(t, str, "secure")
}
