package cloudapi

type RequestConfig struct {
	//API 支持的协议类型，可以多选，多选情况下以英文逗号隔开，如：”HTTP,HTTPS”，取值为：HTTP、HTTPS
	RequestProtocol string

	//HTTP Method，取值为：GET、POST、DELETE、PUT、HEADER、TRACE、PATCH、CONNECT、OPTIONS
	RequestHttpMethod string

	//	API path，比如API的完全地址为http://api.a.com：8080/object/add?key1=value1&key2=value2，path是指/object/add这一部分
	RequestPath	string

	//	请求的模式，取值为：MAPPING、PASSTHROUGH，分别表示入参映射、入参透传
	RequestMode string

	// 	POST/PUT请求时，表示数据以何种方式传递给服务器，取值为：FORM、STREAM，分别表示表单形式(k-v对应)、字节流形式。当RequestMode值为MAPPING时有效。
	BodyFormat string

	//	Body描述
	PostBodyDescription	string
}

type RequestParameter struct {
	// 	String	参数名
	ApiParameterName string

	// 	String 参数位置，取值为：BODY、HEAD、QUERY、PATH
	Location string

	// 	String	参数类型，取值为：String、Int、Long、Float、Double、Boolean，分别表示字符、整型、长整型、单精度浮点型、双精度浮点型、布尔
	ParameterType string

	//	String	是否必填，取值为：REQUIRED、OPTIONAL，分别表示必填、不必填
	Required string

	// 	String	默认值
	DefaultValue string

	// 	String	示例
	DemoValue string

	//	Long	当ParameterType=Int、Long、Float、Double，参数的最大值限定
	MaxValue int64

	// 	Long	当ParameterType=Int、Long、Float、Double，参数的最小值限定
	MinValue int64

	// 	Long	当ParameterType=String，参数的最大长度限定
	MaxLength int

	// 	Long	当ParameterType=String，参数的最小长度限定
	MinLength int

	// 	String	当ParameterType=String，参数验证(正则表达式)
	RegularExpression string

	// 	String	当ParameterType=String，JSON验证(Json Scheme)
	JsonScheme string

	// 	String	当ParameterType=Int、Long、Float、Double或String，允许输入的散列值，不同的值用英文的逗号分隔，形如：1,2,3,4,9或A,B,C,E,F
	EnumValue string

	// 	String	文档可见，取值为：PUBLIC、PRIVATE
	DocShow string

	// 	Integer	文档中顺序
	DocOrder int

	// 	String	参数描述
	Description int
}

type ServiceConfig struct {
	// 	后端服务协议类型，目前只支持HTTP/HTTPS
	ServiceProtocol	string

	// 	调用后端服务地址，比如后端服务完全地址为http://api.a.com:8080/object/add?key1=value1&key2=value2，ServiceAddress是指http://api.a.com:8080这一部分
	ServiceAddress	string

	// 	调用后端服务path，比如后端服务完全地址为http://api.a.com:8080/object/add?key1=value1&key2=value2，ServicePath是指/object/add这一部分
	ServicePath	string

	// 	调用后端服务HTTP协议时的Method，取值为：GET、POST、DELETE、PUT、HEADER、TRACE、PATCH、CONNECT、OPTIONS
	ServiceHttpMethod string

	// 	后端服务超时时间，单位：毫秒
	ServiceTimeout	string

	//	调用后端服务HTTP服务时，ContentType头的取值策略：
	// 	DEFAULT：使用API网关默认的值
	// 	CUSTOM：自定义
	// 	CLIENT：使用客户端上行的ContentType的头
	ContentTypeCatagory	string

	//	调用后端服务HTTP服务，ContentTypeCatagory的值为DEFAULT或者CUSTOM时，ContentType头的取值
	ContentTypeValue	string

	// 	是否采取Mock模式，目前可以取值：
	// TRUE：启用Mock模式
	// FALSE：不启用Mock模式
	Mock	string

	// 	如果启用Mock模式，返回的结果
	MockResult	string

	// Aone
	AoneAppName string

	// 是否启用VPC通道，目前可以取值：
	// TRUE：启用VPC通道
	// FALSE：不启用VPC通
	// 必须先添加VPC授权成功后才能启用
	ServiceVpcEnable	string

	// 如果启用VPC通道，VPC通道相关配置项，详情见ApiAttributesType.md#VpcConfig
	VpcConfig	VpcConfig
}


type VpcConfig struct {
	VpcId string
	InstanceId string
	Port int
}
