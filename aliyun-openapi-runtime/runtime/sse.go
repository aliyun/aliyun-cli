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
	"strings"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/dara"
)

// SSEEvent is one normalized JSON object emitted for an SSE frame.
type SSEEvent json.RawMessage

func sendSSE(ec *ExecContext, req *AssembledRequest, yield func(SSEEvent)) error {
	call, err := prepareCall(ec, req)
	if err != nil {
		return err
	}

	LogRequest(req)
	events := make(chan *openapiClient.SSEResponse, 1)
	errs := make(chan error, 1)
	go call.client.CallSSEApi(call.params, call.request, call.runtime, events, errs)

	// CallSSEApi/ReadAsSSE send their terminal error on errs while producing
	// events. Drain the event channel first so the producer can always finish,
	// then read the terminal error exactly once.
	for response := range events {
		if response == nil {
			continue
		}
		if event, ok := MarshalSSEEvent(response.Event); ok && yield != nil {
			yield(event)
		}
	}
	if err, ok := <-errs; ok && err != nil {
		return fmt.Errorf("runtime: SSE call failed: %w", err)
	}
	return nil
}

// MarshalSSEEvent matches aliyun-cli-runtime's event shape: id/event/retry
// are included when present, while data is decoded as JSON when possible and
// otherwise retained as a string. Empty frames are skipped.
func MarshalSSEEvent(event *dara.SSEEvent) (SSEEvent, bool) {
	if event == nil {
		return nil, false
	}
	object := make(map[string]any)
	if event.Id != nil && *event.Id != "" {
		object["id"] = *event.Id
	}
	if event.Event != nil && *event.Event != "" {
		object["event"] = *event.Event
	}
	if event.Retry != nil {
		object["retry"] = *event.Retry
	}
	if event.Data != nil && *event.Data != "" {
		raw := strings.TrimSpace(*event.Data)
		var parsed any
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			object["data"] = parsed
		} else {
			object["data"] = raw
		}
	}
	if len(object) == 0 {
		return nil, false
	}
	data, err := json.Marshal(object)
	if err != nil {
		return nil, false
	}
	return SSEEvent(data), true
}
