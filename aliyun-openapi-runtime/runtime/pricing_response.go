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
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PricingBusinessError signals a failed quote whose full server response has
// already been written to stdout. Hosts must not print a second error document.
type PricingBusinessError struct {
	Code, Message, RequestID string
}

func (e *PricingBusinessError) Error() string {
	return fmt.Sprintf("cost estimation failed: %s: %s (requestId: %s)", e.Code, e.Message, e.RequestID)
}
func (*PricingBusinessError) ExitCode() int               { return 3 }
func (*PricingBusinessError) RenderError(io.Writer) error { return nil }

// PricingResponseError classifies the unfiltered server document. Version 2
// uses top-level success. Legacy servers remain readable without synthesizing
// a new response or losing their request IDs and messages.
func PricingResponseError(raw string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil || fields == nil {
		return fmt.Errorf("invalid pricing response: expected a JSON object")
	}
	stringField := func(key string) string {
		var value string
		_ = json.Unmarshal(fields[key], &value)
		return value
	}
	if _, v2 := fields["code"]; v2 {
		var success *bool
		if err := json.Unmarshal(fields["success"], &success); err != nil || success == nil ||
			strings.TrimSpace(stringField("code")) == "" || strings.TrimSpace(stringField("requestId")) == "" {
			return fmt.Errorf("invalid pricing response: success, code and requestId are required")
		}
		var message string
		if err := json.Unmarshal(fields["message"], &message); err != nil || string(fields["message"]) == "null" {
			return fmt.Errorf("invalid pricing response: message must be a string")
		}
		price, present := fields["price"]
		// POP omits null fields for outcomes without a quote. Preserve that
		// shape; a successful quote still needs its price data.
		if !present && stringField("code") == "Success" {
			return fmt.Errorf("invalid pricing response: successful quote requires price")
		}
		if present && string(price) != "null" && (len(price) == 0 || price[0] != '{') {
			return fmt.Errorf("invalid pricing response: price must be an object or null")
		}
		if !*success {
			return &PricingBusinessError{Code: stringField("code"), Message: message, RequestID: stringField("requestId")}
		}
		return nil
	}
	// POP's legacy error shape (HTTP 400) and a direct legacy backend shape.
	code, message, requestID, _ := pricingErrorFields([]byte(raw))
	if code == pricingCodeNotRequired {
		return nil
	}
	if code == pricingCodeNotSupported {
		return &PricingBusinessError{Code: code, Message: message, RequestID: requestID}
	}
	var legacy struct {
		Success      *bool  `json:"success"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		RequestID    string `json:"upstreamRequestId"`
	}
	price := fields["price"]
	if price == nil {
		price = json.RawMessage(raw)
	}
	if err := json.Unmarshal(price, &legacy); err != nil || legacy.Success == nil {
		return fmt.Errorf("invalid pricing response: no recognized quote outcome")
	}
	if !*legacy.Success {
		return &PricingBusinessError{Code: legacy.ErrorCode, Message: legacy.ErrorMessage, RequestID: legacy.RequestID}
	}
	return nil
}
