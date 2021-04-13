package lib

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"os"
	"strings"
)

var specChineseHash = SpecText{

	synopsisText: "计算本地文件的crc64或md5",

	paramText: "file_url [options]",

	syntaxText: ` 
    ossutil hash file_url [--type=hashtype]
`,

	detailHelpText: ` 
    该命令计算本地文件的crc64值或md5/content-md5值, 可以通过--type选项来控制计算的类型，
    可选类型值为crc64或md5, 默认为` + DefaultHashType + `。

    注意：oss文件的crc64和content-md5值一般可通过stat命令查看到，参考` + StatCRC64 + `
    字段和` + StatContentMD5 + `字段。若文件在oss支持crc64功能之前上传，则stat命令不支持查看crc64值；
    对于append和multipart类型的文件，stat命令不支持查看content-md5值。

    crc64的计算标准参考ECMA-182标准(http://www.ecma-international.org/publications/standards/Ecma-182.htm)。

    计算类型为md5时，会同时输出文件的md5以及content-md5值。content-md5值其实是先计算md5
    值获得128比特位数字，然后对该数字进行base64编码得到的值。关于content-md5的更多信息，
    请参考https://tools.ietf.org/html/rfc1864。

用法:

    ossutil hash file_url [--type=hashtype] 
`,

	sampleText: ` 
    1) 计算本地文件的crc64: 
        ossutil hash test.txt 或 
        ossutil hash test.txt --type=crc64
   
        输出:
        CRC64-ECMA                  : 295992936743767023        
       
    2) 计算本地文件的md5: 
        ossutil hash test.txt --type=md5

        输出:
        MD5                         : 01C3C45C03B2AF225EFAD9F911A33D73
        Content-MD5                 : AcPEXAOyryJe+tn5EaM9cw==
`,
}

var specEnglishHash = SpecText{

	synopsisText: "Get crc64 or md5 of local file",

	paramText: "file_url [options]",

	syntaxText: ` 
    ossutil hash file_url [--type=hashtype]
`,

	detailHelpText: ` 
    The command calculate crc64 or md5/content-md5 value of the specified local file, 
    specify the hashtype by --type, default hashtype is ` + DefaultHashType + `. 

    Warning: user can use stat command to check the crc64 or md5/content-md5 value of 
    normal oss object, see the ` + StatCRC64 + ` and ` + StatContentMD5 + ` field. If the object 
    was uploaded to oss before oss support crc64 feature, stat result will not show 
    ` + StatCRC64 + `, if the object is append file type or multipart, stat result 
    will not show ` + StatContentMD5 + `. 

    Crc64 is calcuated according to ECMA-182(http://www.ecma-international.org/publications/standards/Ecma-182.htm).

    When hashtype is md5, it will output both md5 and content-md5 of local file. 
    Content-md5 is base64 encoded string of md5. For more detial about content-md5, 
    please refer to https://tools.ietf.org/html/rfc1864.

Usage:

    ossutil hash file_url [--type=hashtype] 
`,

	sampleText: ` 
    1) Get crc64 of local file: 
        ossutil hash test.txt or
        ossutil hash test.txt --type=crc64
        
        output: 
        CRC64-ECMA                  : 295992936743767023        

    2) Get md5  of local file: 
        ossutil hash test.txt --type=md5
         
        output:
        MD5                         : 01C3C45C03B2AF225EFAD9F911A33D73
        Content-MD5                 : AcPEXAOyryJe+tn5EaM9cw==
`,
}

// HashCommand is the command to get crc64/md5 of local file
type HashCommand struct {
	command Command
}

var hashCommand = HashCommand{
	command: Command{
		name:        "hash",
		nameAlias:   []string{""},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseHash,
		specEnglish: specEnglishHash,
		group:       GroupTypeAdditionalCommand,
		validOptionNames: []string{
			OptionHashType,
			OptionLogLevel,
		},
	},
}

// function for RewriteLoadConfiger interface
func (hc *HashCommand) rewriteLoadConfig(configFile string) error {
	// read config file, if error exist, do not print error
	var err error
	if hc.command.configOptions, err = LoadConfig(configFile); err != nil {
		hc.command.configOptions = OptionMapType{}
	}
	return nil
}

// function for FormatHelper interface
func (hc *HashCommand) formatHelpForWhole() string {
	return hc.command.formatHelpForWhole()
}

func (hc *HashCommand) formatIndependHelp() string {
	return hc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (hc *HashCommand) Init(args []string, options OptionMapType) error {
	return hc.command.Init(args, options, hc)
}

// RunCommand simulate inheritance, and polymorphism
func (hc *HashCommand) RunCommand() error {
	hashType, _ := GetString(OptionHashType, hc.command.options)
	path := hc.command.args[0]

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Seek(0, os.SEEK_SET)

	switch strings.ToLower(hashType) {
	case MD5HashType:
		return hashMD5(f)
	default:
		return hashCRC64(f)
	}
}

func hashMD5(f io.Reader) error {
	md5Ins := md5.New()
	w, _ := md5Ins.(hash.Hash)
	if _, err := io.Copy(w, f); err != nil {
		return err
	}

	result := md5Ins.Sum(nil)
	fmt.Printf("%-28s: %X\n", HashMD5, result)

	encoded := base64.StdEncoding.EncodeToString(result)
	fmt.Printf("%-28s: %s\n", HashContentMD5, encoded)
	return nil
}

func hashCRC64(f io.Reader) error {
	crc64Ins := crc64.New(crc64.MakeTable(crc64.ECMA))
	w, _ := crc64Ins.(hash.Hash)
	if _, err := io.Copy(w, f); err != nil {
		return err
	}

	result := crc64Ins.Sum64()
	fmt.Printf("%-28s: %d\n", HashCRC64, result)
	return nil
}
