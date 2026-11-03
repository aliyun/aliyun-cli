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

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// PricingParameterInput carries the prepared API parameters without depending
// on a particular SDK request type. Later slots override earlier slots.
type PricingParameterInput struct {
	Query, Form, Path map[string]string
	Body              any
	Region            string
}

func BuildPricingParameters(input PricingParameterInput) (map[string]any, error) {
	parameters := make(map[string]any)
	for _, slot := range []map[string]string{input.Query, input.Form, input.Path} {
		for key, value := range slot {
			if key != "" && value != "" {
				parameters[key] = value
			}
		}
	}
	if err := MergePricingBody(parameters, input.Body); err != nil {
		return nil, err
	}
	if _, exists := parameters["RegionId"]; !exists && input.Region != "" {
		parameters["RegionId"] = input.Region
	}
	return parameters, nil
}

// MergePricingBody keeps nested JSON values and exact numbers intact. Raw
// --body/--body-file input must be a single JSON object.
func MergePricingBody(parameters map[string]any, body any) error {
	var fields map[string]any
	switch value := body.(type) {
	case nil:
		return nil
	case map[string]any:
		fields = value
	case string:
		return MergePricingBody(parameters, []byte(value))
	case []byte:
		if len(value) == 0 {
			return nil
		}
		decoder := json.NewDecoder(bytes.NewReader(value))
		decoder.UseNumber()
		if err := decoder.Decode(&fields); err != nil || fields == nil {
			return fmt.Errorf("--estimate-cost requires the request body to be a JSON object")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return fmt.Errorf("--estimate-cost requires the request body to be a single JSON object")
		}
	default:
		return fmt.Errorf("--estimate-cost cannot read the request body of type %T; expected a JSON object", body)
	}
	for key, value := range fields {
		parameters[key] = value
	}
	return nil
}
