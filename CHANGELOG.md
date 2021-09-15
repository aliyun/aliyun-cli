# CHANGE LOG
Newest change logs published with release description at https://github.com/aliyun/aliyun-cli/releases . This file just display change logs before version v3.0.1.

### v3.0.0 GA Version

- refactoring cli package design, support composite flag with fields
- refactoring openapi package design, make it more extensible
- support `--quiet` flag
- support `--dryrun` flag
- support `aliyun oss --profile xxx`

### 0.81

- support `--output`
- support `--waiter`
- use `go -ldflags` to enable single Version in Makefile

### 0.80

- support auto completion for zsh/bash
- fix bugs for RepeatList parameter
- refactoring RamRoleArn and EcsRamRole authenticate flow
- oss command can support RamRoleArn and EcsRamRole authenticate mode
- oss command can support --profile and other configure flags

### 0.70

- integrate `ossutil` toolset with aliyun-cli
- optimize `--help` command messages
- config flags (such as ak, profile, sts) can used with openapi call
- support `configure delete`
- fix bug with restful force call

### 0.61

- support --all-pages flags to merge pager APIs

### 0.60

- support suggestions
- optimized error and help message
- integrate more completion of metadata
- fix some caller bugs

### 0.50

- support i18n `aliyun-openapi-meta`
- full support `configure [get|set|list]` command
- optimize help
- support `--quiet` flag

### 0.33

- fix bug for error processing when rpc/restful call
- auto add Content-Type header for restful call

### 0.32

- auto migrate legacy settings

### 0.31

- fix bug of check parameters, skip Action, Region parameters
- support `aliyun configure list` command

### 0.30

- integrate with 64 products meta
- implemented help command for product and api
- support fully certificated mode AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair,

### 0.16

- support --content-type flag to set Header
- support --body-file flat to use file as body input

### 0.15

- support ecs-ram-role
- fix cross platform build problem
- test after configure

### 0.12

- fix bug for configure
- ignore case of ProductName

### 0.11

- Support simple ROA call

### 0.1

- Refactoring with golang
- Basic configure
- Auto endpoint locator
- 2018.1.11