package cloudapi

type SwaggerEntity struct {
	Version string 	`json:"version",yaml:"version"`
	Info SwaggerInfo	`json:"info"`
	Host string 	`json:"host"`
	BasePath string 	`json:"basePath"`
	Schemes []string	`json:"schemes"`
	Paths map[string]SwaggerPath	`json:"paths"`
	Definitions map[string]SwaggerDefinition `json:"definitions"`
	AliyunVpcId string `json:"x-aliyun-vpc-id"`
	AliyunInstanceId string `json:"x-aliyun-instance-id"`
	AliyunPort int `json:"x-aliyun-instance-port"`
}

type SwaggerInfo struct {
	Version string 	`json:"version"`
	Title string 	`json:"title"`
}

type SwaggerPath map[string]SwaggerMethod

type SwaggerMethod struct {
	Consumes []string	`json:"consumes"`
	Produces []string 	`json:"produces"`
	Parameters []SwaggerParameter `json:"parameters"`
	Responses map[string]SwaggerResponse `json:"responses"`
}

type SwaggerParameter struct {
	Name string `json:"name"`
	In string	`json:"in"`
	Description string `json:"description"`
	Required bool `json:"required"`
	Schema SwaggerDefinition `json:"schema"`
}

type SwaggerResponse struct {
	Description string 	`json:"description"`
	Schema SwaggerDefinition `json:"schema"`
}

type SwaggerDefinition struct {
	Ref string			`json:"$ref"`
	Type string 		`json:"type"`
	Required []string 	`json:"required"`
	Properties map[string]SwaggerProperty `json:"properties"`
}

type SwaggerProperty struct {
	Type string 	`json:"type"`
	MinLength int 	`json:"minLength"`
	MaxLength int 	`json:"maxLength"`
	Pattern string 	`json:"pattern"`
	Format string 	`json:"format"`
	Enums []string 	`json:"enum"`
}


