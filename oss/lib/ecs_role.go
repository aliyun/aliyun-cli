package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const (
	AdvanceSeconds int64 = 60
)

type STSAkJson struct {
	AccessKeyId     string `json:"AccessKeyId,omitempty"`
	AccessKeySecret string `json:"AccessKeySecret,omitempty"`
	SecurityToken   string `json:"SecurityToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
	LastUpDated     string `json:"LastUpDated,omitempty"`
	Code            string `json:"Code,omitempty"`
}

func (stsJson *STSAkJson) String() string {
	return fmt.Sprintf("AccessKeyId:%s,AccessKeySecret:%s,SecurityToken:%s,Expiration:%s,LastUpDated:%s",
		stsJson.AccessKeyId, stsJson.AccessKeySecret, stsJson.SecurityToken, stsJson.Expiration, stsJson.LastUpDated)
}

type EcsRoleAK struct {
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
}

func (ecsRole *EcsRoleAK) GetAccessKeyID() string {
	return ecsRole.AccessKeyId
}

func (ecsRole *EcsRoleAK) GetAccessKeySecret() string {
	return ecsRole.AccessKeySecret
}

func (ecsRole *EcsRoleAK) GetSecurityToken() string {
	return ecsRole.SecurityToken
}

// for ecs bind ram and get ak by ossutil automaticly
type EcsRoleAKBuild struct {
	lock            sync.Mutex
	HasGet          bool
	url             string //url for get ak,such as http://100.100.100.200/latest/meta-data/Ram/security-credentials/RamRoleName
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      string
	LastUpDated     string
}

func (roleBuild *EcsRoleAKBuild) GetCredentials() oss.Credentials {
	cred, _ := roleBuild.GetCredentialsE()
	return cred
}

func (roleBuild *EcsRoleAKBuild) GetCredentialsE() (oss.Credentials, error) {
	roleBuild.lock.Lock()
	defer roleBuild.lock.Unlock()

	akJson := STSAkJson{}
	var err error = nil
	bTimeOut := false

	if !roleBuild.HasGet {
		bTimeOut = true
	} else {
		bTimeOut = roleBuild.IsTimeOut()
	}

	if bTimeOut {
		tStart := time.Now().UnixNano() / 1000 / 1000
		akJson, err = roleBuild.HttpReqAk()
		tEnd := time.Now().UnixNano() / 1000 / 1000

		if err != nil {
			return &EcsRoleAK{}, err
		} else {
			roleBuild.AccessKeyId = akJson.AccessKeyId
			roleBuild.AccessKeySecret = akJson.AccessKeySecret
			roleBuild.SecurityToken = akJson.SecurityToken
			roleBuild.Expiration = akJson.Expiration
			LogInfo("get sts ak success,%s,cost:%d(ms)\n", akJson.String(), tEnd-tStart)
		}
	}
	return &EcsRoleAK{
		AccessKeyId:     roleBuild.AccessKeyId,
		AccessKeySecret: roleBuild.AccessKeySecret,
		SecurityToken:   roleBuild.SecurityToken,
	}, nil
}

func (roleBuild *EcsRoleAKBuild) IsTimeOut() bool {
	if roleBuild.Expiration == "" {
		return false
	}

	// attention: can't use time.ParseInLocation(),ecsRole.Expiration is UTC time
	utcExpirationTime, _ := time.Parse("2006-01-02T15:04:05Z", roleBuild.Expiration)

	// Now() returns the current local time
	nowLocalTime := time.Now()

	// Unix() returns the number of seconds elapsedsince January 1, 1970 UTC.
	if utcExpirationTime.Unix()-nowLocalTime.Unix()-AdvanceSeconds <= 0 {
		return true
	}
	return false
}

func (roleBuild *EcsRoleAKBuild) HttpReqAk() (STSAkJson, error) {
	akJson := STSAkJson{}

	//http time out
	c := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := c.Get(roleBuild.url)
	if err != nil {
		LogError("insight getAK,http client get error,url is %s,%s\n", roleBuild.url, err.Error())
		return akJson, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return akJson, err
	}

	err = json.Unmarshal(body, &akJson)
	if err != nil {
		LogError("insight getAK,json.Unmarshal error,body is %s,%s\n", string(body), err.Error())
		return akJson, err
	}

	// parsar json,such as
	//{
	//    "AccessKeyId" : "XXXXXXXXX",
	//    "AccessKeySecret" : "XXXXXXXXX",
	//    "Expiration" : "2017-11-01T05:20:01Z",
	//    "SecurityToken" : "XXXXXXXXX",
	//    "LastUpdated" : "2017-10-31T23:20:01Z",
	//    "Code" : "Success"
	// }

	if akJson.Code != "" && strings.ToUpper(akJson.Code) != "SUCCESS" {
		LogError("insight getAK,get sts ak error,code:%s\n", akJson.Code)
		return akJson, fmt.Errorf("insight getAK,get sts ak error,code:%s", akJson.Code)
	}

	if akJson.AccessKeyId == "" || akJson.AccessKeySecret == "" {
		LogError("insight getAK,parsar http json body error:\n%s\n", string(body))
		return akJson, fmt.Errorf("insight getAK,parsar http json body error:\n%s\n", string(body))
	}

	if akJson.Expiration != "" {
		_, err := time.Parse("2006-01-02T15:04:05Z", akJson.Expiration)
		if err != nil {
			LogError("time.Parse error,Expiration is %s,%s\n", akJson.Expiration, err.Error())
			return akJson, err
		}
	}

	roleBuild.HasGet = true
	return akJson, nil
}
