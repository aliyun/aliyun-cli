package meta

import "fmt"

type InvalidEndpointError struct {
	Region string
	Product *Product
}

func (e *InvalidEndpointError) Error() string {
	s := fmt.Sprintf("unknown endpoint for region %s", e.Region)
	//if e.Product != "" {
	//	s = s + fmt", try add --endpoint %s", e.Suggestion
	//}
	// TODO add suggestion
	return s
}


