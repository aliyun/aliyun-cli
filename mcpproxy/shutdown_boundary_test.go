package mcpproxy

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

type shutdownListener struct {
	connections chan net.Conn
	closeOnce   sync.Once
	closeErr    error
}

func (l *shutdownListener) Accept() (net.Conn, error) {
	conn, ok := <-l.connections
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (l *shutdownListener) Close() error {
	l.closeOnce.Do(func() { close(l.connections) })
	return l.closeErr
}

func (*shutdownListener) Addr() net.Addr { return &net.TCPAddr{} }

func TestMCPShutdownClosesRequestsAfterGracePeriod(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		proxy := testProxy(t)
		started, finished := make(chan struct{}), make(chan struct{})
		proxy.Server = proxy.newHTTPServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			close(started)
			<-req.Context().Done()
			close(finished)
		}))
		listener := &shutdownListener{connections: make(chan net.Conn, 1)}
		client, server := net.Pipe()
		defer client.Close()
		listener.connections <- server
		served := make(chan error, 1)
		go func() { served <- proxy.Server.Serve(listener) }()
		_, err := client.Write([]byte("GET / HTTP/1.1\r\nHost: proxy.test\r\n\r\n"))
		require.NoError(t, err)
		<-started
		start := time.Now()
		require.NoError(t, proxy.Stop())
		require.Equal(t, 10*time.Second, time.Since(start))
		require.ErrorIs(t, <-served, http.ErrServerClosed)
		<-finished
	})
}

func TestMCPShutdownPreservesListenerError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		proxy := testProxy(t)
		proxy.Server = proxy.newHTTPServer(http.NewServeMux())
		cause := errors.New("listener close failed")
		listener := &shutdownListener{connections: make(chan net.Conn), closeErr: cause}
		served := make(chan error, 1)
		go func() { served <- proxy.Server.Serve(listener) }()
		synctest.Wait()
		require.ErrorIs(t, proxy.Stop(), cause)
		require.ErrorIs(t, <-served, http.ErrServerClosed)
	})
}
