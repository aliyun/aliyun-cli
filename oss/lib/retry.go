package lib

import (
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Total attempts retain the existing retry-times/retry-count meaning. Only
// reads can replay an ambiguous transport/5xx failure. A 429 explicitly rejects
// the request and is safe to retry; arbitrary local errors are never retried.
func retryOSS(err error, attempt int, limit int64, readOnly bool) bool {
	if err == nil || int64(attempt) >= limit || !retryableOSS(err, readOnly) {
		return false
	}
	retrySleep(retryDelay(attempt))
	return true
}

var retrySleep = time.Sleep

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	ceiling := time.Duration(1<<uint(attempt-1)) * 200 * time.Millisecond
	return ceiling/2 + time.Duration(rand.Int64N(int64(ceiling/2)+1))
}
func retryableOSS(err error, readOnly bool) bool {
	var service oss.ServiceError
	found := errors.As(err, &service)
	if !found {
		var p *oss.ServiceError
		if errors.As(err, &p) && p != nil {
			service = *p
			found = true
		}
	}
	if found {
		return service.StatusCode == 429 || (readOnly && service.StatusCode >= 500 && service.StatusCode <= 599)
	}
	var refresh *credentialRefreshError
	if errors.As(err, &refresh) {
		return false
	}
	if !readOnly {
		return false
	}
	var network net.Error
	return errors.As(err, &network) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}
