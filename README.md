# Aliyun Command Line Interface

[中文文档](./README_cn.md)

This is a refactoring beta version rewrite with golang, for old stable python verison switch to [python]() branch

## Overview

Aliyun Command Line Interface `aliyun` is a unified tool to manage your Aliyun services. Using this tool you can easily invoke the Aliyun OpenAPI to control multiple Aliyun services from the command line and also automate them through scripts, for instance using the Bash shell or Python.

## Install 

You can download binary release in the following links 

- [Mac](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-macosx-0.30-amd64.tgz)
- [Linux](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-linux-0.30-amd64.tgz)
- [Windows](http://aliyun-cli.oss-cn-hangzhou.aliyuncs.com/aliyun-cli-windows-0.30-amd64.tgz)

OR

clone this repo, make it by youself by `make`


## Configure

Before using `aliyun` you should create a AccessKeyId/Secret from your console. 

```
$ aliyuncli configure
Aliyun Access Key ID [None]: <Your aliyun access key id>
Aliyun Access Key Secret [None]: <Your aliyun access key secret>
Default Region Id [None]: cn-hangzhou
Default output format [json]: table
```


## Usage

### Basic Usage

```
$ aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...
```

Here are some examples: :

```
$ aliyun rds DescribeDBInstances --PageSize 50
$ aliyun ecs DescribeRegions
$ aliyun rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx
```

### Additional Flags


- `--profile`  use configured profile
- `--force`    call OpenAPI without check
- `--header`   add custom HTTP header with --header x-foo=bar
- `--endpoint` use assigned endpoint
- `--region`   use assigned region
- `--version`  assign product version

## Command Completion

On Unix-like systems, the `aliyun` includes a command-completion feature that enables you to use the `TAB` key to complete a partially typed command. This feature is not automatically installed, so you need to configure it manually.

Configuring command completion requires two pieces of information:

-   the name of the shell you are using
-   the location of `aliyun_completer` script.

### Check Your Shell

Currently `aliyuncli` supports these shells:

-   bash
-   zsh.

1. To find the `aliyun_completer`, you can use: :

    $ which aliyun_completer
    /usr/local/bin/aliyun_completer

1.  To enable command completion:

bash - use the build-in command complete: :

    $ complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli

zsh - source bin/aliyun\_zsh\_completer.sh :

    % source /usr/local/bin/aliyun_zsh_completer.sh

### Test Command Completion

    $ aliyuncli s<TAB>
    ecs     rds     slb

The services display the SDK(s) you installed.

Finally, to ensure that completion continues to work after a reboot, add a configuration command to enable command completion to your shell profile. :

    $ vim ~/.bash_profile

Add `complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli` at the end of the file.

## Support Products

...





