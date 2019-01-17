package meta

import (
	"github.com/stretchr/testify/assert"

	"fmt"
	"testing"
)

func TestInvalidEndpointError_Error(t *testing.T) {
	err := &InvalidEndpointError{
		Product: &Product{
			RegionalEndpointPattern: "endpoint",
		},
		Region: "cn-hangzhou",
	}
	err.LocationError = fmt.Errorf("here is a error.")
	msg := err.Error()
	assert.Contains(t, msg, "here is a error.")

	err.LocationError = nil
	msg = err.Error()
	assert.Contains(t, msg, "cn-hangzhou")
}
