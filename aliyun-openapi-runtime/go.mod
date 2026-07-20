module github.com/aliyun/aliyun-openapi-runtime

go 1.23.0

require (
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.12
	github.com/alibabacloud-go/tea v1.3.12
	github.com/aliyun/credentials-go v1.4.7
	github.com/jmespath/go-jmespath v0.4.0
	google.golang.org/protobuf v1.26.0
)

// Mirror the main module's pin so standalone builds resolve the same
// jmespath fork.
replace github.com/jmespath/go-jmespath => github.com/alibabacloud-go/go-jmespath v0.4.0-aliyun

require (
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	golang.org/x/net v0.26.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
