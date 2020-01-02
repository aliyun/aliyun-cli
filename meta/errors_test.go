// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
