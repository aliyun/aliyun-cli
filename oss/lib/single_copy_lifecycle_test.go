package lib

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestSingleObjectTransfersWaitForStatistics(t *testing.T) {
	for _, operation := range []string{"download", "copy"} {
		for _, failure := range []bool{false, true} {
			name := operation + "/success"
			if failure {
				name = operation + "/transfer_failure"
			}
			t.Run(name, func(t *testing.T) {
				transferred := make(chan struct{})
				var transferOnce sync.Once
				var heads, transfers atomic.Int32
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					w.Header().Set("Connection", "close")
					if request.Method == http.MethodHead {
						heads.Add(1)
						w.Header().Set("Content-Length", "7")
						w.Header().Set("Last-Modified", "Thu, 01 Oct 2026 00:00:00 GMT")
						return
					}
					transfers.Add(1)
					defer transferOnce.Do(func() { close(transferred) })
					if operation == "download" {
						require.Equal(t, http.MethodGet, request.Method)
					} else {
						require.Equal(t, http.MethodPut, request.Method)
						require.Equal(t, "/bucket/key", request.Header.Get("X-Oss-Copy-Source"))
					}
					if failure {
						w.WriteHeader(http.StatusForbidden)
						_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>transfer denied</Message><RequestId>local-request</RequestId></Error>`)
					} else if operation == "download" {
						_, _ = io.WriteString(w, "content")
					} else {
						_, _ = io.WriteString(w, `<CopyObjectResult><LastModified>2026-10-01T00:00:00.000Z</LastModified><ETag>test-etag</ETag></CopyObjectResult>`)
					}
				}))
				server.Config.SetKeepAlivesEnabled(false)
				server.Start()
				defer server.Close()
				destination := filepath.Join(t.TempDir(), "download")
				previousProgress, previousSignals := chProgressSignal, atomic.LoadInt64(&signalNum)
				defer func() {
					chProgressSignal = previousProgress
					atomic.StoreInt64(&signalNum, previousSignals)
				}()
				synctest.Test(t, func(t *testing.T) {
					scanStarted, releaseScan := make(chan struct{}), make(chan struct{})
					var scanOnce sync.Once
					statisticsFunction := runtime.FuncForPC(reflect.ValueOf((*CopyCommand).objectStatistic).Pointer()).Name()
					trace := &httptrace.ClientTrace{GetConn: func(string) {
						// Both callers issue the same HEAD request. Gate the statistics
						// caller before dialing, independently of request arrival order.
						var callers [32]uintptr
						frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
						for {
							frame, more := frames.Next()
							if frame.Function == statisticsFunction {
								scanOnce.Do(func() { close(scanStarted) })
								<-releaseScan
								return
							}
							if !more {
								return
							}
						}
					}}
					options := []oss.Option{oss.WithContext(httptrace.WithClientTrace(context.Background(), trace))}
					command := CopyCommand{command: localOSSCommand(t, server.URL)}
					command.cpOption.force, command.cpOption.threshold = true, 1024
					command.cpOption.options, command.cpOption.payerOptions = options, options
					command.monitor.init(operationTypeGet)
					if operation == "copy" {
						command.monitor.init(operationTypeCopy)
					}
					chProgressSignal = make(chan chProgressSignalType, 10)
					atomic.StoreInt64(&signalNum, 0)
					finished := make(chan error, 1)
					go func() {
						source := CloudURL{bucket: "bucket", object: "key"}
						if operation == "download" {
							finished <- command.downloadFiles(source, FileURL{urlStr: destination})
						} else {
							finished <- command.copyFiles(source, CloudURL{bucket: "other", object: "copy"})
						}
					}()
					<-scanStarted
					<-transferred
					synctest.Wait()
					var result error
					returnedEarly := false
					select {
					case result = <-finished:
						returnedEarly = true
						t.Error("transfer returned before its statistics request completed")
					default:
					}
					close(releaseScan)
					if !returnedEarly {
						result = <-finished
					}
					synctest.Wait()
					if failure {
						var serviceError oss.ServiceError
						require.ErrorAs(t, result, &serviceError)
						require.Equal(t, "AccessDenied", serviceError.Code)
					} else {
						require.NoError(t, result)
						if operation == "download" {
							content, err := os.ReadFile(destination)
							require.NoError(t, err)
							require.Equal(t, "content", string(content))
						}
					}
					require.True(t, command.monitor.seekAheadEnd)
					require.EqualValues(t, 1, command.monitor.totalNum)
					require.EqualValues(t, 7, command.monitor.totalSize)
				})
				require.EqualValues(t, 2, heads.Load())
				require.EqualValues(t, 1, transfers.Load())
			})
		}
	}
}
