package cloudapi

import (
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cloudapi"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/cli"
)

func getClient(prof *config.Profile) (*cloudapi.Client, error) {
	sc, err := prof.GetSessionCredential()
	if err != nil {
		return nil, err
	}
	if sc.StsToken != "" {
		return cloudapi.NewClientWithStsToken(prof.RegionId, sc.AccessKeyId, sc.AccessKeySecret, sc.StsToken)
	} else {
		return cloudapi.NewClientWithAccessKey(prof.RegionId, sc.AccessKeyId, sc.AccessKeySecret)
	}
}

func getApiName(basePath string, path string, method string) string {
	return method +
		strings.Replace(basePath, "/", "_", -1) +
		strings.Replace(path, "/", "_", -1)
}

func getApiSample(se *SwaggerEntity) {
	return

}

func parseHost2(s string) (protocol string, address string, path string, ok bool) {
	if strings.HasPrefix(s, "http://") {
		protocol = "http"
		address, path, ok = cli.SplitStringWithPrefix(s[len("http://"):], "/")
	} else if strings.HasPrefix(s, "https://") {
		protocol = "https"
		address, path, ok = cli.SplitStringWithPrefix(s[len("https://"):], "/")
	} else {
		ok = false
	}
	return
}

func removePrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	} else {
		return s, false
	}
}