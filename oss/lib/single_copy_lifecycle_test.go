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
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestSingleObjectTransfersReuseSourceMetadata(t *testing.T) {
	const sourceModified = "Thu, 01 Oct 2026 00:00:00 GMT"
	for _, operation := range []string{"download", "copy"} {
		for _, scenario := range []string{"success", "transfer_failure", "skip", "range"} {
			if operation == "copy" && scenario == "range" {
				continue
			}
			t.Run(operation+"/"+scenario, func(t *testing.T) {
				failure := scenario == "transfer_failure"
				wantContent, wantSize := "content", int64(7)
				if scenario == "range" {
					wantContent, wantSize = "ont", 3
				}
				var heads, transfers atomic.Int32
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
					w.Header().Set("Connection", "close")
					if request.Method == http.MethodHead {
						require.Equal(t, "/bucket/key", request.URL.Path)
						if request.URL.Path == "/bucket/key" {
							heads.Add(1)
						}
						w.Header().Set("Content-Length", "7")
						w.Header().Set("Last-Modified", sourceModified)
						return
					}
					transfers.Add(1)
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
						if scenario == "range" {
							require.Equal(t, "bytes=1-3", request.Header.Get("Range"))
							w.Header().Set("Content-Range", "bytes 1-3/7")
							w.WriteHeader(http.StatusPartialContent)
						}
						_, _ = io.WriteString(w, wantContent)
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
					releaseScan := make(chan struct{})
					var statisticsRequests atomic.Int32
					statisticsFunction := runtime.FuncForPC(reflect.ValueOf((*CopyCommand).objectStatistic).Pointer()).Name()
					trace := &httptrace.ClientTrace{GetConn: func(string) {
						// Keep an obsolete statistics request blocked before dialing so
						// it cannot hide behind a fast response or request arrival order.
						var callers [32]uintptr
						frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
						for {
							frame, more := frames.Next()
							if frame.Function == statisticsFunction {
								statisticsRequests.Add(1)
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
					if scenario == "skip" {
						modified, err := time.Parse(http.TimeFormat, sourceModified)
						require.NoError(t, err)
						command.cpOption.startTime = modified.Unix() + 1
					}
					if scenario == "range" {
						command.cpOption.vrange = "1-3"
					}
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
					synctest.Wait()
					var result error
					returned := false
					select {
					case result = <-finished:
						returned = true
					default:
						t.Error("single-object transfer waited for a redundant statistics request")
					}
					if count := statisticsRequests.Load(); count != 0 {
						t.Errorf("independent statistics requests = %d, want 0", count)
					}
					close(releaseScan)
					if !returned {
						result = <-finished
					}
					synctest.Wait()
					if failure {
						var serviceError oss.ServiceError
						require.ErrorAs(t, result, &serviceError)
						require.Equal(t, "AccessDenied", serviceError.Code)
					} else {
						require.NoError(t, result)
						if operation == "download" && scenario != "skip" {
							content, err := os.ReadFile(destination)
							require.NoError(t, err)
							require.Equal(t, wantContent, string(content))
						}
					}
					require.True(t, command.monitor.seekAheadEnd)
					require.EqualValues(t, 1, command.monitor.totalNum)
					require.EqualValues(t, wantSize, command.monitor.totalSize)
					snapshot := command.monitor.getSnapshot()
					require.EqualValues(t, 1, snapshot.fileNum+snapshot.skipNum+snapshot.errNum)
					if failure {
						require.EqualValues(t, 1, snapshot.errNum)
					} else if scenario == "skip" {
						require.EqualValues(t, 1, snapshot.skipNum)
					} else {
						require.EqualValues(t, 1, snapshot.fileNum)
					}
					if !failure {
						require.EqualValues(t, wantSize, snapshot.dealSize)
					}
				})
				require.EqualValues(t, 1, heads.Load())
				if scenario == "skip" {
					require.Zero(t, transfers.Load())
				} else {
					require.EqualValues(t, 1, transfers.Load())
				}
			})
		}
	}
}
