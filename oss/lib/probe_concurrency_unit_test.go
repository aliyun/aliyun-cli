package lib

import (
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestProbeBandwidthCalibrationUsesRealStagesWithVirtualTime(t *testing.T) {
	const transferBytes = 1024
	for _, mode := range []string{"upload-speed", "download-speed"} {
		t.Run(mode, func(t *testing.T) {
			output, err := os.Create(filepath.Join(t.TempDir(), "calibration.log"))
			require.NoError(t, err)
			previousOutput := os.Stdout
			os.Stdout = output
			t.Cleanup(func() { os.Stdout = previousOutput; output.Close() })
			synctest.Test(t, func(t *testing.T) {
				var requests, active, maximum atomic.Int32
				var uploaded atomic.Int64
				client := &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
					requests.Add(1)
					current := active.Add(1)
					defer active.Add(-1)
					for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
					}
					time.Sleep(100 * time.Millisecond)
					if err := r.Context().Err(); err != nil {
						return nil, err
					}
					var checksum uint64
					if r.Method == http.MethodPut {
						payload := make([]byte, transferBytes)
						n, err := io.ReadFull(r.Body, payload)
						if err != nil {
							return nil, err
						}
						uploaded.Add(int64(n))
						checksum = crc64.Checksum(payload, oss.CrcTable())
						if err := r.Body.Close(); err != nil {
							return nil, err
						}
					}
					body := ""
					if r.Method == http.MethodGet {
						body = strings.Repeat("x", transferBytes)
						checksum = crc64.Checksum([]byte(body), oss.CrcTable())
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Length": {fmt.Sprint(len(body))}, "Etag": {`"probe"`}, oss.HTTPHeaderOssCRC64: {strconv.FormatUint(checksum, 10)}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: r}, nil
				})}
				previous := newOSSClient
				newOSSClient = func(endpoint, id, secret string, options ...oss.ClientOption) (*oss.Client, error) {
					return previous(endpoint, id, secret, append(options, oss.HTTPClient(client))...)
				}
				defer func() { newOSSClient = previous }()
				pc := ProbeCommand{command: localOSSCommand(t, "http://oss.test")}
				pc.command.options[OptionDisableCRC64] = boolPointer(false)
				pc.pbOption.bucketName, pc.pbOption.objectName, pc.pbOption.probeItem = "bucket", "key", mode
				pc.command.options[OptionRuntime] = stringPointer("1")
				started := time.Now()
				require.NoError(t, pc.DetectBandWidth())
				require.GreaterOrEqual(t, time.Since(started), time.Minute)
				time.Sleep(time.Second)
				synctest.Wait()
				require.Zero(t, active.Load())
				stacks := make([]byte, 1<<20)
				stacks = stacks[:runtime.Stack(stacks, true)]
				require.NotContains(t, string(stacks), "(*ProbeCommand).PutObjectWithContext")
				require.NotContains(t, string(stacks), "(*ProbeCommand).GetObjectWithContext")
				require.Greater(t, requests.Load(), int32(runtime.NumCPU()*10))
				require.Greater(t, maximum.Load(), int32(runtime.NumCPU()))
				if mode == "upload-speed" {
					require.Greater(t, uploaded.Load(), int64(0))
				}
			})
			data, err := os.ReadFile(output.Name())
			require.NoError(t, err)
			require.Contains(t, string(data), "suggest parallel is")
			require.Contains(t, string(data), "max average speed is")
			_, speed, found := strings.Cut(string(data), "max average speed is ")
			require.True(t, found)
			measured, err := strconv.ParseFloat(strings.Fields(speed)[0][:strings.Index(strings.Fields(speed)[0], "(")], 64)
			require.NoError(t, err)
			require.Greater(t, measured, float64(0))
			_, runtimeOutput, found := strings.Cut(string(data), "run "+mode+"  1 seconds with parallel ")
			require.True(t, found)
			_, runtimeSpeed, found := strings.Cut(runtimeOutput, "average speed:")
			require.True(t, found)
			runtimeMeasured, err := strconv.ParseFloat(runtimeSpeed[:strings.Index(runtimeSpeed, "(")], 64)
			require.NoError(t, err)
			require.Greater(t, runtimeMeasured, float64(0))
		})
	}
}

func TestProbeDownloadPartsJoinWorkersAndProgressOnEveryExit(t *testing.T) {
	for _, scenario := range []string{"success", "request failure", "short body", "single part"} {
		t.Run(scenario, func(t *testing.T) {
			var active atomic.Int32
			ranges := make(chan string, 8)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				active.Add(1)
				defer active.Add(-1)
				if r.Method == http.MethodHead {
					w.Header().Set("Content-Length", "12")
					return
				}
				ranges <- r.Header.Get("Range")
				if scenario == "request failure" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				var start, end int
				if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Length", fmt.Sprint(end-start+1))
				w.WriteHeader(http.StatusPartialContent)
				if scenario == "short body" {
					io.WriteString(w, "x")
					return
				}
				io.WriteString(w, "abcdefghijkl"[start:end+1])
			}))
			defer server.Close()
			pc := ProbeCommand{command: localOSSCommand(t, server.URL)}
			pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "key"
			pc.command.options[OptionParallel], pc.command.options[OptionPartSize] = stringPointer("2"), stringPointer("4")
			if scenario == "single part" {
				pc.command.options[OptionPartSize], pc.command.options[OptionParallel] = stringPointer("0"), stringPointer("0")
			}
			err := pc.DetectDownloadTime()
			if scenario == "success" || scenario == "single part" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			server.Close()
			require.Zero(t, active.Load())
			close(ranges)
			var got []string
			for value := range ranges {
				got = append(got, value)
			}
			sort.Strings(got)
			if scenario == "success" {
				require.Equal(t, []string{"bytes=0-3", "bytes=4-7", "bytes=8-11"}, got)
			} else if scenario == "single part" {
				require.Equal(t, []string{"bytes=0-11"}, got)
			} else {
				require.NotEmpty(t, got)
			}
			stack := make([]byte, 1<<20)
			n := runtime.Stack(stack, true)
			require.NotContains(t, string(stack[:n]), "(*ProbeCommand).DetectDownloadTime.func")
		})
	}
}

type probeSlowBody struct {
	reader *strings.Reader
	closed *atomic.Int32
}

func (b *probeSlowBody) Read(p []byte) (int, error) {
	time.Sleep(time.Second)
	return b.reader.Read(p[:min(len(p), 8192)])
}

func (b *probeSlowBody) Close() error { b.closed.Add(1); return nil }

func TestProbeDownloadTickerStopsAfterCompletedTransfer(t *testing.T) {
	output, err := os.Create(filepath.Join(t.TempDir(), "progress.log"))
	require.NoError(t, err)
	previousOutput := os.Stdout
	os.Stdout = output
	t.Cleanup(func() { os.Stdout = previousOutput; output.Close() })
	synctest.Test(t, func(t *testing.T) {
		var closed atomic.Int32
		client := &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
			var body io.ReadCloser = io.NopCloser(strings.NewReader(""))
			if r.Method == http.MethodGet {
				body = &probeSlowBody{reader: strings.NewReader(strings.Repeat("x", 32768)), closed: &closed}
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Length": {"32768"}}, Body: body, ContentLength: 32768, Request: r}, nil
		})}
		previous := newOSSClient
		newOSSClient = func(endpoint, id, secret string, options ...oss.ClientOption) (*oss.Client, error) {
			return previous(endpoint, id, secret, append(options, oss.HTTPClient(client))...)
		}
		defer func() { newOSSClient = previous }()
		pc := ProbeCommand{command: localOSSCommand(t, "http://oss.test")}
		pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "key"
		pc.command.options[OptionParallel], pc.command.options[OptionPartSize] = stringPointer("1"), stringPointer("0")
		started := time.Now()
		require.NoError(t, pc.DetectDownloadTime())
		require.GreaterOrEqual(t, time.Since(started), 4*time.Second)
		require.EqualValues(t, 1, closed.Load())
		synctest.Wait()
	})
	data, err := os.ReadFile(output.Name())
	require.NoError(t, err)
	require.Contains(t, string(data), "downloading average speed:")
	require.Contains(t, string(data), "total bytes:32768")
}

func TestProbeDownloadFailureCancelsInFlightRequestBeforeReturning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		blocked, canceled := make(chan struct{}), make(chan struct{})
		var active atomic.Int32
		client := &http.Client{Transport: maintenanceTransport(func(r *http.Request) (*http.Response, error) {
			active.Add(1)
			defer active.Add(-1)
			if r.Method == http.MethodHead {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Length": {"8"}}, Body: io.NopCloser(strings.NewReader("")), ContentLength: 8, Request: r}, nil
			}
			if r.Header.Get("Range") == "bytes=0-3" {
				close(blocked)
				<-r.Context().Done()
				close(canceled)
				return nil, r.Context().Err()
			}
			<-blocked
			return &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`<Error><Code>AccessDenied</Code></Error>`)), Request: r}, nil
		})}
		previous := newOSSClient
		newOSSClient = func(endpoint, id, secret string, options ...oss.ClientOption) (*oss.Client, error) {
			return previous(endpoint, id, secret, append(options, oss.HTTPClient(client))...)
		}
		defer func() { newOSSClient = previous }()
		pc := ProbeCommand{command: localOSSCommand(t, "http://oss.test")}
		pc.pbOption.bucketName, pc.pbOption.objectName = "bucket", "key"
		pc.command.options[OptionParallel], pc.command.options[OptionPartSize] = stringPointer("2"), stringPointer("4")
		require.ErrorContains(t, pc.DetectDownloadTime(), "AccessDenied")
		select {
		case <-canceled:
		default:
			t.Fatal("the in-flight request was not canceled before return")
		}
		require.Zero(t, active.Load())
		synctest.Wait()
	})
}
