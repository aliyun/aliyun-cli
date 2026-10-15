package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBucketCnameAndReplicationLocalRequests(t *testing.T) {
	for _, tc := range []struct {
		name, method, item, verb, query string
		args                            []string
		newCommand                      func(Command) Commander
	}{
		{"cname-get", "get", "", "GET", "cname", nil, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"cname-put", "put", "", "POST", "cname", []string{"example.test"}, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"cname-delete", "delete", "", "POST", "cname", []string{"example.test"}, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"token-get", "get", "token", "GET", "comp=token", []string{"example.test"}, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"token-put", "put", "token", "POST", "comp=token", []string{"example.test"}, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"certificate", "put", "certificate", "POST", "comp=add", []string{"$file"}, func(c Command) Commander { return &BucketCnameCommand{command: c} }},
		{"replication-put", "put", "", "POST", "replication", []string{"$file"}, func(c Command) Commander { return &ReplicationCommand{command: c} }},
		{"replication-get", "get", "", "GET", "replication", nil, func(c Command) Commander { return &ReplicationCommand{command: c} }},
		{"replication-delete", "delete", "", "POST", "replication", []string{"rule"}, func(c Command) Commander { return &ReplicationCommand{command: c} }},
		{"replication-location", "get", "location", "GET", "replicationLocation", nil, func(c Command) Commander { return &ReplicationCommand{command: c} }},
		{"replication-progress", "get", "progress", "GET", "replicationProgress", []string{"rule"}, func(c Command) Commander { return &ReplicationCommand{command: c} }},
		{"replication-rtc", "put", "rtc", "PUT", "rtc", []string{"$file"}, func(c Command) Commander { return &ReplicationCommand{command: c} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var failed atomic.Bool
			requests := make(chan string, 8)
			body := `<CnameToken><Bucket>bucket</Bucket><Cname>example.test</Cname><Token>local-token</Token></CnameToken>`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r.Method + " " + r.URL.RawQuery
				if failed.Load() {
					w.WriteHeader(http.StatusForbidden)
					io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
					return
				}
				io.WriteString(w, body)
			}))
			defer server.Close()
			base := localOSSCommand(t, server.URL)
			base.options[OptionMethod], base.options[OptionItem] = &tc.method, &tc.item
			file := filepath.Join(t.TempDir(), "config")
			require.NoError(t, os.WriteFile(file, []byte(`<Configuration><Rule>rule</Rule></Configuration>`), 0600))
			base.args = []string{"oss://bucket"}
			for _, arg := range tc.args {
				if arg == "$file" {
					arg = file
				}
				base.args = append(base.args, arg)
			}
			require.NoError(t, tc.newCommand(base).RunCommand())
			request := <-requests
			require.True(t, strings.HasPrefix(request, tc.verb+" "), request)
			require.Contains(t, request, tc.query)
			failed.Store(true)
			require.ErrorContains(t, tc.newCommand(base).RunCommand(), "AccessDenied")
			<-requests
			base.options[OptionEndpoint] = stringPointer("")
			require.ErrorContains(t, tc.newCommand(base).RunCommand(), "endpoint")
		})
	}
}

func TestBucketCnameTokensWriteAndConfirmFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<CnameToken><Bucket>bucket</Bucket><Cname>example.test</Cname><Token>local-token</Token></CnameToken>`)
	}))
	defer server.Close()
	for _, method := range []string{"get", "put"} {
		for _, token := range []bool{false, true} {
			if method == "put" && !token {
				continue
			}
			base := localOSSCommand(t, server.URL)
			base.options[OptionMethod] = &method
			base.args = []string{"oss://bucket"}
			if token {
				base.options[OptionItem] = stringPointer("token")
				base.args = append(base.args, "example.test")
			}
			output := filepath.Join(t.TempDir(), "token")
			base.args = append(base.args, output)
			run := func() { require.NoError(t, (&BucketCnameCommand{command: base}).RunCommand()) }
			run()
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			require.Contains(t, string(data), "local-token")
			require.NoError(t, os.WriteFile(output, []byte("original"), 0600))
			withBucketInput(t, "n\n", run)
			data, err = os.ReadFile(output)
			require.NoError(t, err)
			require.Equal(t, "original", string(data))
			withBucketInput(t, "y\n", run)
			data, err = os.ReadFile(output)
			require.NoError(t, err)
			require.Contains(t, string(data), "local-token")
			base.args[len(base.args)-1] = filepath.Join(output, "bad")
			require.Error(t, (&BucketCnameCommand{command: base}).RunCommand())
		}
	}
}

func TestBucketInventoryPaginationAndStyleLists(t *testing.T) {
	var pages atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("style") {
			io.WriteString(w, `<StyleList><Style><Name>thumbnail</Name></Style></StyleList>`)
		} else {
			pages.Add(1)
			if r.URL.Query().Get("continuation-token") == "" {
				io.WriteString(w, `<ListInventoryConfigurationsResult><IsTruncated>true</IsTruncated><NextContinuationToken>next</NextContinuationToken><InventoryConfiguration><Id>first</Id></InventoryConfiguration></ListInventoryConfigurationsResult>`)
			} else {
				io.WriteString(w, `<ListInventoryConfigurationsResult><IsTruncated>false</IsTruncated><InventoryConfiguration><Id>second</Id></InventoryConfiguration></ListInventoryConfigurationsResult>`)
			}
		}
	}))
	defer server.Close()
	base := localOSSCommand(t, server.URL)
	base.options[OptionMethod] = stringPointer("list")
	output := filepath.Join(t.TempDir(), "inventory")
	base.args = []string{"oss://bucket", output}
	cmd := &BucketInventoryCommand{command: base}
	require.NoError(t, cmd.RunCommand())
	require.EqualValues(t, 2, pages.Load())
	require.Equal(t, 2, cmd.bwOption.ruleCount)
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(data), "first")
	require.Contains(t, string(data), "second")
	base.args[1] = filepath.Join(t.TempDir(), "styles")
	require.NoError(t, (&BucketStyleCommand{command: base}).RunCommand())
	data, err = os.ReadFile(base.args[1])
	require.NoError(t, err)
	require.Contains(t, string(data), "thumbnail")
}

func TestBucketWormLocalLifecycle(t *testing.T) {
	var denied atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if denied.Load() {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `<Error><Code>AccessDenied</Code></Error>`)
			return
		}
		w.Header().Set("X-Oss-Worm-Id", "worm-local")
		if r.Method == http.MethodGet {
			io.WriteString(w, `<WormConfiguration><WormId>worm-local</WormId><State>Locked</State><RetentionPeriodInDays>30</RetentionPeriodInDays></WormConfiguration>`)
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	for _, args := range [][]string{{"init", "oss://bucket", "30"}, {"abort", "oss://bucket"}, {"complete", "oss://bucket", "worm-local"}, {"extend", "oss://bucket", "60", "worm-local"}, {"get", "oss://bucket"}} {
		base := localOSSCommand(t, server.URL)
		base.args = args
		cmd := &WormCommand{command: base}
		require.NoError(t, cmd.RunCommand())
		if args[0] == "get" {
			require.Equal(t, "worm-local", cmd.wmOption.wormConfig.WormId)
		}
		denied.Store(true)
		require.ErrorContains(t, (&WormCommand{command: base}).RunCommand(), "AccessDenied")
		denied.Store(false)
		base.options[OptionEndpoint] = stringPointer("")
		require.ErrorContains(t, (&WormCommand{command: base}).RunCommand(), "endpoint")
	}
	for _, args := range [][]string{{"bad", "oss://bucket"}, {"get", "local"}, {"get", "oss://"}, {"init", "oss://bucket"}, {"init", "oss://bucket", "bad"}, {"complete", "oss://bucket"}, {"extend", "oss://bucket", "10"}, {"extend", "oss://bucket", "bad", "id"}} {
		base := localOSSCommand(t, server.URL)
		base.args = args
		require.Error(t, (&WormCommand{command: base}).RunCommand())
	}
}
