package credentials

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alibabacloud-go/debug/debug"
	"github.com/aliyun/credentials-go/credentials/request"
	"github.com/aliyun/credentials-go/credentials/response"
	"github.com/aliyun/credentials-go/credentials/utils"
)

var debuglog = debug.Init("credential")

var hookParse = func(err error) error {
	return err
}

// Credential is an interface for getting actual credential
type Credential interface {
	GetAccessKeyID() (string, error)
	GetAccessSecret() (string, error)
	GetSecurityToken() (string, error)
	GetBearerToken() string
	GetType() string
}

// Configuration is important when call NewCredential
type Configuration struct {
	Type                  string `json:"type"`
	AccessKeyID           string `json:"access_key_id"`
	AccessKeySecret       string `json:"access_key_secret"`
	RoleArn               string `json:"role_arn"`
	RoleSessionName       string `json:"role_session_name"`
	PublicKeyID           string `json:"public_key_id"`
	RoleName              string `json:"role_name"`
	SessionExpiration     int    `json:"session_expiration"`
	PrivateKeyFile        string `json:"private_key_file"`
	BearerToken           string `json:"bearer_token"`
	SecurityToken         string `json:"security_token"`
	RoleSessionExpiration int    `json:"role_session_expiratioon"`
	Policy                string `json:"policy"`
	Host                  string `json:"host"`
	Timeout               int    `json:"timeout"`
	ConnectTimeout        int    `json:"connect_timeout"`
	Proxy                 string `json:"proxy"`
}

// NewCredential return a credential according to the type in config.
// if config is nil, the function will use default provider chain to get credential.
// please see README.md for detail.
func NewCredential(config *Configuration) (credential Credential, err error) {
	if config == nil {
		config, err = defaultChain.resolve()
		if err != nil {
			return
		}
		return NewCredential(config)
	}
	switch config.Type {
	case "access_key":
		err = checkAccessKey(config)
		if err != nil {
			return
		}
		credential = newAccessKeyCredential(config.AccessKeyID, config.AccessKeySecret)
	case "sts":
		err = checkSTS(config)
		if err != nil {
			return
		}
		credential = newStsTokenCredential(config.AccessKeyID, config.AccessKeySecret, config.SecurityToken)
	case "ecs_ram_role":
		err = checkEcsRAMRole(config)
		if err != nil {
			return
		}
		runtime := &utils.Runtime{
			Host:           config.Host,
			Proxy:          config.Proxy,
			ReadTimeout:    config.Timeout,
			ConnectTimeout: config.ConnectTimeout,
		}
		credential = newEcsRAMRoleCredential(config.RoleName, runtime)
	case "ram_role_arn":
		err = checkRAMRoleArn(config)
		if err != nil {
			return
		}
		runtime := &utils.Runtime{
			Host:           config.Host,
			Proxy:          config.Proxy,
			ReadTimeout:    config.Timeout,
			ConnectTimeout: config.ConnectTimeout,
		}
		credential = newRAMRoleArnCredential(config.AccessKeyID, config.AccessKeySecret, config.RoleArn, config.RoleSessionName, config.Policy, config.RoleSessionExpiration, runtime)
	case "rsa_key_pair":
		err = checkRSAKeyPair(config)
		if err != nil {
			return
		}
		file, err1 := os.Open(config.PrivateKeyFile)
		if err1 != nil {
			err = fmt.Errorf("InvalidPath: Can not open PrivateKeyFile, err is %s", err1.Error())
			return
		}
		defer file.Close()
		var privateKey string
		scan := bufio.NewScanner(file)
		for scan.Scan() {
			if strings.HasPrefix(scan.Text(), "----") {
				continue
			}
			privateKey += scan.Text() + "\n"
		}
		runtime := &utils.Runtime{
			Host:           config.Host,
			Proxy:          config.Proxy,
			ReadTimeout:    config.Timeout,
			ConnectTimeout: config.ConnectTimeout,
		}
		credential = newRsaKeyPairCredential(privateKey, config.PublicKeyID, config.SessionExpiration, runtime)
	case "bearer":
		if config.BearerToken == "" {
			err = errors.New("BearerToken cannot be empty")
			return
		}
		credential = newBearerTokenCredential(config.BearerToken)
	default:
		err = errors.New("Invalid type option, support: access_key, sts, ecs_ram_role, ram_role_arn, rsa_key_pair")
		return
	}
	return credential, nil
}

func checkRSAKeyPair(config *Configuration) (err error) {
	if config.PrivateKeyFile == "" {
		err = errors.New("PrivateKeyFile cannot be empty")
		return
	}
	if config.PublicKeyID == "" {
		err = errors.New("PublicKeyID cannot be empty")
		return
	}
	return
}

func checkRAMRoleArn(config *Configuration) (err error) {
	if config.AccessKeySecret == "" {
		err = errors.New("AccessKeySecret cannot be empty")
		return
	}
	if config.RoleArn == "" {
		err = errors.New("RoleArn cannot be empty")
		return
	}
	if config.RoleSessionName == "" {
		err = errors.New("RoleSessionName cannot be empty")
		return
	}
	if config.AccessKeyID == "" {
		err = errors.New("AccessKeyID cannot be empty")
		return
	}
	return
}

func checkEcsRAMRole(config *Configuration) (err error) {
	if config.RoleName == "" {
		err = errors.New("RoleName cannot be empty")
		return
	}
	return
}

func checkSTS(config *Configuration) (err error) {
	if config.AccessKeyID == "" {
		err = errors.New("AccessKeyID cannot be empty")
		return
	}
	if config.AccessKeySecret == "" {
		err = errors.New("AccessKeySecret cannot be empty")
		return
	}
	if config.SecurityToken == "" {
		err = errors.New("SecurityToken cannot be empty")
		return
	}
	return
}

func checkAccessKey(config *Configuration) (err error) {
	if config.AccessKeyID == "" {
		err = errors.New("AccessKeyID cannot be empty")
		return
	}
	if config.AccessKeySecret == "" {
		err = errors.New("AccessKeySecret cannot be empty")
		return
	}
	return
}

func doAction(request *request.CommonRequest, runtime *utils.Runtime) (content []byte, err error) {
	httpRequest, err := http.NewRequest(request.Method, request.URL, strings.NewReader(""))
	if err != nil {
		return
	}
	httpRequest.Proto = "HTTP/1.1"
	httpRequest.Host = request.Domain
	debuglog("> %s %s %s", httpRequest.Method, httpRequest.URL.RequestURI(), httpRequest.Proto)
	debuglog("> Host: %s", httpRequest.Host)
	for key, value := range request.Headers {
		if value != "" {
			debuglog("> %s: %s", key, value)
			httpRequest.Header[key] = []string{value}
		}
	}
	debuglog(">")
	httpClient := &http.Client{}
	httpClient.Timeout = time.Duration(runtime.ReadTimeout) * time.Second
	proxy := &url.URL{}
	if runtime.Proxy != "" {
		proxy, err = url.Parse(runtime.Proxy)
		if err != nil {
			return
		}
	}
	trans := &http.Transport{}
	if proxy != nil && runtime.Proxy != "" {
		trans.Proxy = http.ProxyURL(proxy)
	}
	trans.DialContext = utils.Timeout(time.Duration(runtime.ConnectTimeout) * time.Second)
	httpClient.Transport = trans
	httpResponse, err := hookDo(httpClient.Do)(httpRequest)
	if err != nil {
		return
	}
	debuglog("< %s %s", httpResponse.Proto, httpResponse.Status)
	for key, value := range httpResponse.Header {
		debuglog("< %s: %v", key, strings.Join(value, ""))
	}
	debuglog("<")

	resp := &response.CommonResponse{}
	err = hookParse(resp.ParseFromHTTPResponse(httpResponse))
	if err != nil {
		return
	}
	debuglog("%s", resp.GetHTTPContentString())
	if resp.GetHTTPStatus() != http.StatusOK {
		err = fmt.Errorf("httpStatus: %d, message = %s", resp.GetHTTPStatus(), resp.GetHTTPContentString())
		return
	}
	return resp.GetHTTPContentBytes(), nil
}
