# cloudapi_importSwagger (alpha)

## Usage

- create a Api Group with API Gateway console, got `ApiGroupId`
- create a VPC Access with API Gateway console, modify swagger schema with following sample
    - use vpc access, you need use `x-aliyun-vpc-id`, `x-aliyun-instance-id`, `x-aliyun-instance-port` extensions

```json
{
  "swagger": "2.0",
  "info": {
    "version": "1.0.0",
    "title": "Test"
  },
  "host": "http://192.168.0.132",
  "basePath": "/Dev",
  "schemes": [
    "https"
  ],
  "x-aliyun-vpc-id": "your-vpc-id",
  "x-aliyun-instance-id":  "your-instance-id",
  "x-aliyun-instance-port": your-vpc-port,
  ...
}
```

## Use aliyun cli to import swagger

Download cli from links: (3.0.3)

- [Mac](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-macosx-3.0.3-amd64.tgz)
- [Linux](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-linux-3.0.3-amd64.tgz)
- [Windows (64 bit)](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-windows-3.0.3-amd64.zip)

Configure cli tools with configure command.

```
$ aliyun configure
```

use following command to import swagger:

```
$ aliyun cloudapi_importSwagger path/to/swagger.json --GroupId <ApiGroupId>
```

run `cloudapi_importSwagger` twice will update you api settings, if you wan't to delete imported apis, use `--delete-all` flag

```
$ aliyun cloudapi_importSwagger path/to/swagger.json --GroupId <ApiGroupId> --delete-all
```

