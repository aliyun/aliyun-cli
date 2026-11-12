package legacyapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ServerError preserves the PascalCase error presentation. Kebab/Tea errors
// deliberately keep their separate existing formatter.
type ServerError struct {
	RespHeaders                                          http.Header
	status                                               int
	code, message, requestID, hostID, recommend, comment string
	accessDeniedDetail                                   map[string]interface{}
}

func NewServerError(status int, body, comment string) error {
	e := &ServerError{status: status, message: body, comment: comment}
	var data map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &data) == nil {
		for key, target := range map[string]*string{"Code": &e.code, "Message": &e.message, "RequestId": &e.requestID, "HostId": &e.hostID, "Recommend": &e.recommend} {
			if raw, ok := data[key]; ok && string(raw) != "null" {
				_ = json.Unmarshal(raw, target)
			}
		}
		_ = json.Unmarshal(data["AccessDeniedDetail"], &e.accessDeniedDetail)
	}
	return e
}
func (e *ServerError) Error() string {
	out := fmt.Sprintf("SDK.ServerError\nErrorCode: %s\nRecommend: %s\nRequestId: %s\nMessage: %s\nRespHeaders: %s", e.code, e.comment+e.recommend, e.requestID, e.message, map[string][]string(e.RespHeaders))
	if e.accessDeniedDetail != nil {
		out += fmt.Sprintf("\nAccessDeniedDetail: %s", e.accessDeniedDetail)
	}
	return out
}
func (e *ServerError) ErrorCode() string                          { return e.code }
func (e *ServerError) Message() string                            { return e.message }
func (e *ServerError) RequestId() string                          { return e.requestID }
func (e *ServerError) HostId() string                             { return e.hostID }
func (e *ServerError) HttpStatus() int                            { return e.status }
func (e *ServerError) Recommend() string                          { return e.recommend }
func (e *ServerError) AccessDeniedDetail() map[string]interface{} { return e.accessDeniedDetail }

func (e *ServerError) explainSignature(stringToSign string) {
	if stringToSign == "" || (e.code != "SignatureDoesNotMatch" && e.code != "IncompleteSignature") {
		return
	}
	_, server, ok := strings.Cut(e.message, "server string to sign is:")
	if !ok {
		return
	}
	if server == stringToSign {
		e.recommend = "InvalidAccessKeySecret: Please check you AccessKeySecret"
	} else {
		e.recommend = "This may be a bug with the SDK and we hope you can submit this question in the github issue(https://github.com/aliyun/alibaba-cloud-sdk-go/issues), thanks very much"
	}
}

type ClientError struct {
	code, message string
	cause         error
}

func NewClientError(code, message string, cause error) error {
	return &ClientError{code, message, cause}
}
func (e *ClientError) Error() string {
	out := fmt.Sprintf("[%s] %s", e.code, e.message)
	if e.cause != nil {
		out += "\ncaused by:\n" + e.cause.Error()
	}
	return out
}
func (e *ClientError) OriginError() error { return e.cause }
func (e *ClientError) Unwrap() error      { return e.cause }
