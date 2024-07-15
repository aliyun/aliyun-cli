package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketInventory = SpecText{
	synopsisText: "添加、查询、删除或者列举bucket的清单配置",

	paramText: "bucket_url [local_xml_file] [id] [options]",

	syntaxText: ` 
	ossutil inventory --method put oss://bucket local_xml_file [options]
    ossutil inventory --method get oss://bucket id [local_file] [options]
    ossutil inventory --method delete oss://bucket id [options]
    ossutil inventory --method list oss://bucket [local_file] [--marker marker] [options]
`,
	detailHelpText: ` 
    inventory命令通过设置method选项值为put、get、delete、list,可以添加、查询、删除、列举bucket的清单配置

用法:
    该命令有四种用法:
	
    1) ossutil inventory --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取清单配置,然后添加一个bucket的清单规则
        配置文件是一个xml格式的文件,如果已经存在标识为配置文件中id的配置,则报错
        下面是一个配置文件例子
   
        <?xml version="1.0" encoding="UTF-8"?>
        <InventoryConfiguration>
            <Id>report1</Id>
            <IsEnabled>true</IsEnabled>
            <Filter>
                <Prefix>filterPrefix/</Prefix>
            </Filter>
            <Destination>
                <OSSBucketDestination>
                    <Format>CSV</Format>
                    <AccountId>123456789012</AccountId>
                    <RoleArn>acs:ram::1287905056319499:role/fsr7hs5tjnxkiepp2ka6</RoleArn>
                    <Bucket>acs:oss:::destination-bucket</Bucket>
                    <Prefix>prefix1</Prefix>
                    <Encryption>
                        <SSE-KMS>
                            <KeyId>keyId</KeyId>
                        </SSE-KMS>
                    </Encryption>
                </OSSBucketDestination>
            </Destination>
            <Schedule>
                <Frequency>Daily</Frequency>
            </Schedule>
            <IncludedObjectVersions>All</IncludedObjectVersions>
            <OptionalFields>
                <Field>Size</Field>
                <Field>LastModifiedDate</Field>
                <Field>ETag</Field>
                <Field>StorageClass</Field>
                <Field>IsMultipartUploaded</Field>
                <Field>EncryptionStatus</Field>
            </OptionalFields>
        </InventoryConfiguration>
      
    2) ossutil inventory --method get oss://bucket id [local_xml_file] [options]
        这个命令查询bucket的标识为id的清单配置
        如果输入参数local_xml_file，清单配置将输出到该文件，否则输出到屏幕上
	
    3) ossutil inventory --method delete oss://bucket id [options]
        这个命令删除bucket的标识为id的清单配置
    
    4) ossutil inventory --method list oss://bucket [local_file] [--marker marker] [options]
        这个命令列举bucket的清单配置
`,
	sampleText: ` 
    1) 添加bucket的inventory配置
       ossutil inventory --method put oss://bucket local_xml_file

    2) 查询bucket的标识为id的inventory配置，结果输出到标准输出
       ossutil inventory --method get oss://bucket id
	
    3) 删除bucket的标识为id的inventory配置
       ossutil inventory --method delete oss://bucket id
    
    4) 列举bucket的所有inventory配置
       ossutil inventory --method list oss://bucket
`,
}

var specEnglishBucketInventory = SpecText{
	synopsisText: "Add, get, delete, or list bucket inventory configuration",

	paramText: "bucket_url [local_xml_file] [id] [options]",

	syntaxText: ` 
	ossutil inventory --method put oss://bucket local_xml_file [options]
    ossutil inventory --method get oss://bucket id [local_file] [options]
    ossutil inventory --method delete oss://bucket id [options]
    ossutil inventory --method list oss://bucket [local_file] [--marker marker] [options]
`,
	detailHelpText: ` 
    inventory command can add, get, delete or list the inventory configuration of the oss bucket by
    set method option value to put, get, delete, list

Usage:
    There are four usages for this command:
	
    1) ossutil inventory --method put oss://bucket local_xml_file [options]
        The command adds the inventory configuration of bucket from local file local_xml_file
        the local_xml_file is xml format, if there is exist same id of configuration, an error occurs.
        The following is an example xml file

        <?xml version="1.0" encoding="UTF-8"?>
        <InventoryConfiguration>
            <Id>report1</Id>
            <IsEnabled>true</IsEnabled>
            <Filter>
                <Prefix>filterPrefix/</Prefix>
            </Filter>
            <Destination>
                <OSSBucketDestination>
                    <Format>CSV</Format>
                    <AccountId>123456789012</AccountId>
                    <RoleArn>acs:ram::1287905056319499:role/fsr7hs5tjnxkiepp2ka6</RoleArn>
                    <Bucket>acs:oss:::destination-bucket</Bucket>
                    <Prefix>prefix1</Prefix>
                    <Encryption>
                        <SSE-KMS>
                            <KeyId>keyId</KeyId>
                        </SSE-KMS>
                    </Encryption>
                </OSSBucketDestination>
            </Destination>
            <Schedule>
                <Frequency>Daily</Frequency>
            </Schedule>
            <IncludedObjectVersions>All</IncludedObjectVersions>
            <OptionalFields>
                <Field>Size</Field>
                <Field>LastModifiedDate</Field>
                <Field>ETag</Field>
                <Field>StorageClass</Field>
                <Field>IsMultipartUploaded</Field>
                <Field>EncryptionStatus</Field>
            </OptionalFields>
        </InventoryConfiguration>
        
    2) ossutil inventory --method get oss://bucket id [local_xml_file] [options]
       The command gets the inventory configuration of bucket, The identifier of the inventory is id
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3) ossutil inventory --method delete oss://bucket id [options]
       The command deletes the inventory configuration of bucket, The identifier of the inventory is id
      
    4) ossutil inventory --method list oss://bucket [local_file] [--marker marker] [options]
       List the bucket's all inventory configuration
`,
	sampleText: ` 
    1) add bucket inventory
       ossutil inventory --method put oss://bucket local_xml_file

    2) get inventory configuration to stdout, The identifier of the inventory is id
       ossutil inventory --method get oss://bucket id
	
    3) delete inventory configuration, The identifier of the inventory is id
       ossutil inventory --method delete oss://bucket id
    
    4) list the bucket's all inventory configuration
       ossutil inventory --method list oss://bucket
`,
}

type BucketInventoryOptionType struct {
	bucketName string
	ruleCount  int
}

type BucketInventoryCommand struct {
	command  Command
	bwOption BucketInventoryOptionType
}

var bucketInventoryCommand = BucketInventoryCommand{
	command: Command{
		name:        "inventory",
		nameAlias:   []string{"inventory"},
		minArgc:     1,
		maxArgc:     3,
		specChinese: specChineseBucketInventory,
		specEnglish: specEnglishBucketInventory,
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
			OptionMarker,
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
func (bic *BucketInventoryCommand) formatHelpForWhole() string {
	return bic.command.formatHelpForWhole()
}

func (bic *BucketInventoryCommand) formatIndependHelp() string {
	return bic.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bic *BucketInventoryCommand) Init(args []string, options OptionMapType) error {
	return bic.command.Init(args, options, bic)
}

// RunCommand simulate inheritance, and polymorphism
func (bic *BucketInventoryCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bic.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" && strMethod != "list" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete|list")
	}

	srcBucketUrL, err := GetCloudUrl(bic.command.args[0], "")
	if err != nil {
		return err
	}

	bic.bwOption.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bic.PutBucketInventory()
	} else if strMethod == "get" {
		err = bic.GetBucketInventory()
	} else if strMethod == "delete" {
		err = bic.DeleteBucketInventory()
	} else if strMethod == "list" {
		err = bic.ListBucketInventory()
	}
	return err
}

func (bic *BucketInventoryCommand) PutBucketInventory() error {
	if len(bic.command.args) < 2 {
		return fmt.Errorf("put bucket inventory need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := bic.command.args[1]
	fileInfo, err := os.Stat(xmlFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", xmlFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", xmlFile)
	}

	// parsing the xml file
	file, err := os.Open(xmlFile)
	if err != nil {
		return err
	}
	defer file.Close()
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// put bucket inventory
	client, err := bic.command.ossClient(bic.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketInventoryXml(bic.bwOption.bucketName, string(text))
}

func (bic *BucketInventoryCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket inventory: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bic *BucketInventoryCommand) GetBucketInventory() error {
	if len(bic.command.args) < 2 {
		return fmt.Errorf("get bucket inventory need at least 2 parameters,the parameter id is empty")
	}

	inventoryId := bic.command.args[1]

	client, err := bic.command.ossClient(bic.bwOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketInventoryXml(bic.bwOption.bucketName, inventoryId)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bic.command.args) >= 3 {
		fileName := bic.command.args[2]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bic.confirm(fileName)
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

	outFile.Write([]byte(output))

	fmt.Printf("\n\n")

	return nil
}

func (bic *BucketInventoryCommand) DeleteBucketInventory() error {
	if len(bic.command.args) < 2 {
		return fmt.Errorf("delete bucket inventory need at least 2 parameters,the parameter id is empty")
	}

	inventoryId := bic.command.args[1]

	// delete bucket inventory
	client, err := bic.command.ossClient(bic.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.DeleteBucketInventory(bic.bwOption.bucketName, inventoryId)
}

func (bic *BucketInventoryCommand) ListBucketInventory() error {
	bic.bwOption.ruleCount = 0
	vmarker, _ := GetString(OptionMarker, bic.command.options)
	client, err := bic.command.ossClient(bic.bwOption.bucketName)
	if err != nil {
		return err
	}

	var sumResult string
	for {
		xmlBody, err := client.ListBucketInventoryXml(bic.bwOption.bucketName, vmarker)
		if err != nil {
			return err
		}
		sumResult += xmlBody
		sumResult += "\n"

		var result oss.ListInventoryConfigurationsResult
		err = xml.Unmarshal([]byte(xmlBody), &result)
		if err != nil {
			return err
		}

		bic.bwOption.ruleCount += len(result.InventoryConfiguration)

		if result.IsTruncated != nil && *result.IsTruncated {
			vmarker = result.NextContinuationToken
		} else {
			break
		}
	}

	var outFile *os.File
	if len(bic.command.args) >= 2 {
		fileName := bic.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bic.confirm(fileName)
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

	outFile.Write([]byte(sumResult))

	var sumList oss.ListInventoryConfigurationsResult
	err = xml.Unmarshal([]byte(sumResult), &sumList)
	if err != nil {
		return err
	}

	fmt.Printf("\n\ntotal inventory rule count:%d\n", bic.bwOption.ruleCount)

	return nil
}
