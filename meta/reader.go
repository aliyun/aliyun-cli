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
	"encoding/json"
	"fmt"
)

type Reader interface {
	ReadFrom(path string) ([]byte, error)
}

func ReadJsonFrom(r Reader, path string, v interface{}) error {
	buf, err := r.ReadFrom(path)
	if err != nil {
		return fmt.Errorf("read json from %s failed %v", path, v)
	}
	err = json.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("unmarshal json %s failed %v", path, err)
	}
	return nil
}
