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
	"fmt"
	"strings"
)

type InvalidEndpointError struct {
	LocationError error
	Region        string
	Product       *Product
}

func (e *InvalidEndpointError) Error() string {
	s := fmt.Sprintf("unknown endpoint for region %s", e.Region)
	if e.Product != nil {
		//	s = s + fmt", try add --endpoint %s", e.Suggestion
		s = s + "\n  you need to add --endpoint xxx.aliyuncs.com"
		if e.LocationError != nil {
			s = s + fmt.Sprintf("\n  LC_Error: %s", e.LocationError.Error())
		}
		if e.Product.RegionalEndpointPattern != "" {
			ep := strings.Replace(e.Product.RegionalEndpointPattern, "[RegionId]", e.Region, 1)
			s = s + fmt.Sprintf(", sample: --endpoint %s", ep)
		}
	}
	return s
}
