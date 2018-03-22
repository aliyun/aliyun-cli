package meta

import (
	"fmt"
	"strings"
)

type InvalidEndpointError struct {
	LocationError error
	Region string
	Product *Product
}

func (e *InvalidEndpointError) Error() string {
	s := fmt.Sprintf("unknown endpoint for region %s", e.Region)
	if e.Product != nil {
		//	s = s + fmt", try add --endpoint %s", e.Suggestion
		s = s + fmt.Sprintf("\n  you need to add --endpoint xxx.aliyuncs.com")
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


