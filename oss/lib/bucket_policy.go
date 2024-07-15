package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var specChineseBucketPolicy = SpecText{
	synopsisText: "设置、查询或者删除bucket的policy配置",

	paramText: "bucket_url [local_json_file] [options]",

	syntaxText: ` 
	ossutil bucket-policy --method put oss://bucket local_json_file [options]
    ossutil bucket-policy --method get oss://bucket [local_file] [options]
    ossutil bucket-policy --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    bucket-policy命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的policy配置

用法:
    该命令有三种用法:
	
    1) ossutil bucket-policy --method put oss://bucket local_json_file [options]
        这个命令从配置文件local_json_file中读取policy配置,然后设置bucket的policy规则
        配置文件是一个json格式的文件,举例如下
   
        {
            "Version": "1",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": [
                        "ram:ListObjects"
                    ],
                    "Principal": [
                        "1234567"
                    ],
                    "Resource": [
                        "*"
                    ],
                    "Condition": {}
                }
            ]
        }

    2) ossutil bucket-policy --method get oss://bucket [local_json_file] [options]
        这个命令查询bucket的policy配置,如果输入参数local_json_file,policy配置将输出到该文件,否则输出到屏幕上
	
    3) ossutil bucket-policy --method delete oss://bucket [options]
        这个命令删除bucket的policy配置
`,
	sampleText: ` 
    1) 设置bucket的policy配置
       ossutil bucket-policy --method put oss://bucket local_json_file

    2) 查询bucket的policy配置，结果输出到标准输出
       ossutil bucket-policy --method get oss://bucket
	
    3) 查询bucket的policy配置，结果输出到本地文件
       ossutil bucket-policy --method get oss://bucket local_json_file
	
    4) 删除bucket的policy配置
       ossutil bucket-policy --method delete oss://bucket
`,
}

var specEnglishBucketPolicy = SpecText{
	synopsisText: "Set, get or delete bucket policy configuration",

	paramText: "bucket_url [local_json_file] [options]",

	syntaxText: ` 
	ossutil bucket-policy --method put oss://bucket local_json_file [options]
    ossutil bucket-policy --method get oss://bucket [local_json_file] [options]
    ossutil bucket-policy --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    bucket-policy command can set, get and delete the policy configuration of the oss bucket by
    set method option value to put, get, delete

Usage:
    There are three usages for this command:
	
    1) ossutil bucket-policy --method put oss://bucket local_json_file [options]
        The command sets the policy configuration of bucket from local file local_json_file
        the local_json_file is xml format,for example

        {
            "Version": "1",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": [
                        "ram:ListObjects"
                    ],
                    "Principal": [
                        "1234567"
                    ],
                    "Resource": [
                        "*"
                    ],
                    "Condition": {}
                }
            ]
        }
        
    2) ossutil bucket-policy --method get oss://bucket  [local_json_file] [options]
       The command gets the policy configuration of bucket
       If you input parameter local_json_file,the configuration will be output to local_json_file
       If you don't input parameter local_json_file,the configuration will be output to stdout
	
    3) ossutil bucket-policy --method delete oss://bucket [options]
       The command deletes the policy configuration of bucket
`,
	sampleText: ` 
    1) put bucket policy
       ossutil bucket-policy --method put oss://bucket local_json_file

    2) get bucket policy configuration to stdout
       ossutil bucket-policy --method get oss://bucket
	
    3) get bucket policy configuration to local file
       ossutil bucket-policy --method get oss://bucket local_json_file
	
    4) delete bucket policy configuration
       ossutil bucket-policy --method delete oss://bucket
`,
}

type bucketPolicyOptionType struct {
	bucketName string
}

type BucketPolicyCommand struct {
	command  Command
	bpOption bucketPolicyOptionType
}

var bucketPolicyCommand = BucketPolicyCommand{
	command: Command{
		name:        "bucket-policy",
		nameAlias:   []string{"bucket-policy"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketPolicy,
		specEnglish: specEnglishBucketPolicy,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionLogLevel,
			OptionMethod,
			OptionPassword,
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionReadTimeout,
			OptionConnectTimeout,
			OptionSTSRegion,
			OptionSkipVerifyCert,
			OptionUserAgent,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
		},
	},
}

// function for FormatHelper interface
func (bpc *BucketPolicyCommand) formatHelpForWhole() string {
	return bpc.command.formatHelpForWhole()
}

func (bpc *BucketPolicyCommand) formatIndependHelp() string {
	return bpc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bpc *BucketPolicyCommand) Init(args []string, options OptionMapType) error {
	return bpc.command.Init(args, options, bpc)
}

// RunCommand simulate inheritance, and polymorphism
func (bpc *BucketPolicyCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bpc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(bpc.command.args[0], "")
	if err != nil {
		return err
	}

	bpc.bpOption.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bpc.PutBucketPolicy()
	} else if strMethod == "get" {
		err = bpc.GetBucketPolicy()
	} else if strMethod == "delete" {
		err = bpc.DeleteBucketPolicy()
	}
	return err
}

func (bpc *BucketPolicyCommand) PutBucketPolicy() error {
	if len(bpc.command.args) < 2 {
		return fmt.Errorf("put bucket policy need at least 2 parameters,the local json file is empty")
	}

	jsonFile := bpc.command.args[1]
	fileInfo, err := os.Stat(jsonFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", jsonFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", jsonFile)
	}

	// parsing the xml file
	file, err := os.Open(jsonFile)
	if err != nil {
		return err
	}
	defer file.Close()
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// put bucket policy
	client, err := bpc.command.ossClient(bpc.bpOption.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketPolicy(bpc.bpOption.bucketName, string(text))
}

func (bpc *BucketPolicyCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket policy: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bpc *BucketPolicyCommand) GetBucketPolicy() error {
	client, err := bpc.command.ossClient(bpc.bpOption.bucketName)
	if err != nil {
		return err
	}

	policyRes, err := client.GetBucketPolicy(bpc.bpOption.bucketName)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bpc.command.args) >= 2 {
		fileName := bpc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bpc.confirm(fileName)
			if !bConitnue {
				return nil
			}
		}

		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}

	var jsonText bytes.Buffer
	_ = json.Indent(&jsonText, []byte(policyRes), "", "    ")
	outFile.Write(jsonText.Bytes())

	fmt.Printf("\n\n")

	return nil
}

func (bpc *BucketPolicyCommand) DeleteBucketPolicy() error {
	// delete bucket policy
	client, err := bpc.command.ossClient(bpc.bpOption.bucketName)
	if err != nil {
		return err
	}

	return client.DeleteBucketPolicy(bpc.bpOption.bucketName)
}
