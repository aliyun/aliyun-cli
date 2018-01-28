# Aliyun Command Line Interface

[中文文档](./README_cn.md)

This is a refactoring beta version rewrite with golang, for old stable python verison switch to [python]() branch

## Overview

Aliyun Command Line Interface `aliyun` is a unified tool to manage your Aliyun services. Using this tool you can easily invoke the Aliyun OpenAPI to control multiple Aliyun services from the command line and also automate them through scripts, for instance using the Bash shell or Python.

## Install 

You can download binary release in the following links 


OR

make it by youself

## Configure

### 

## Usage

How to Use aliyuncli
--------------------

An `aliyuncli` command has four parts:

-   Name of the tool “aliyuncli”
-   Service name, such as: ecs, rds, slb, ots
-   Available operations for each service
-   List of keys and values, with possible multiple keys and values. The values can be number, string, or JSON format.

Here are some examples: :

    $ aliyuncli rds DescribeDBInstances --PageSize 50
    $ aliyuncli ecs DescribeRegions
    $ aliyuncli rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx

### Additional Usage Information

    --filter

`aliyuncli` supports a filter function. When any API is called, the data returned is JSON formatted by default. The filter function can help the user manipulate the JSON formatted data more easily.

Here are some examples: :

    $ aliyuncli ecs DescribeRegions --output json --filter Regions.Region[0]
    {
        "LocalName":"\u6df1\u5733"
        "RegionId": "cn-shenzhen"
    }
    $ aliyuncli ecs DescribeRegions --output json --filter Regions.Region[*].RegionId
    [
        "cn-shenzhen", 
        "cn-qingdao", 
        "cn-beijing", 
        "cn-hongkong", 
        "cn-hangzhou", 
        "us-west-1"
    ]
    $ aliyuncli ecs DescribeRegions --output json --filter Regions.Region[3].RegionId
    "cn-hongkong"

## Using HTTPS

Your can switch to HTTPS request by add --secure argument to your command.

    $ aliyuncli Ecs DescribeInstances --secure
	

## Command Completion

On Unix-like systems, the `aliyuncli` includes a command-completion feature that enables you to use the `TAB` key to complete a partially typed command. This feature is not automatically installed, so you need to configure it manually.

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

## 

<table style="width:67%;">
<colgroup>
<col width="20%" />
<col width="45%" />
</colgroup>
<thead>
<tr class="header">
<th>Product</th>
<th>SDK</th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>BatchCompute</td>
<td>aliyun-python-sdk-batchcompute</td>
</tr>
<tr class="even">
<td>Bsn</td>
<td>aliyun-python-sdk-bsn</td>
</tr>
<tr class="odd">
<td>Bss</td>
<td>aliyun-python-sdk-bss</td>
</tr>
<tr class="even">
<td>Cms</td>
<td>aliyun-python-sdk-cms</td>
</tr>
<tr class="odd">
<td>Crm</td>
<td>aliyun-python-sdk-crm</td>
</tr>
<tr class="even">
<td>Drds</td>
<td>aliyun-python-sdk-drds</td>
</tr>
<tr class="odd">
<td>Ecs</td>
<td>aliyun-python-sdk-ecs</td>
</tr>
<tr class="even">
<td>Ess</td>
<td>aliyun-python-sdk-ess</td>
</tr>
<tr class="odd">
<td>Ft</td>
<td>aliyun-python-sdk-ft</td>
</tr>
<tr class="even">
<td>Ocs</td>
<td>aliyun-python-sdk-ocs</td>
</tr>
<tr class="odd">
<td>Oms</td>
<td>aliyun-python-sdk-oms</td>
</tr>
<tr class="even">
<td>OssAdmin</td>
<td>aliyun-python-sdk-ossadmin</td>
</tr>
<tr class="odd">
<td>Ram</td>
<td>aliyun-python-sdk-ram</td>
</tr>
<tr class="even">
<td>Ocs</td>
<td>aliyun-python-sdk-ocs</td>
</tr>
<tr class="odd">
<td>Rds</td>
<td>aliyun-python-sdk-rds</td>
</tr>
<tr class="even">
<td>Risk</td>
<td>aliyun-python-sdk-risk</td>
</tr>
<tr class="odd">
<td>R-kvstore</td>
<td>aliyun-python-r-kvstore</td>
</tr>
<tr class="even">
<td>Slb</td>
<td>aliyun-python-sdk-slb</td>
</tr>
<tr class="odd">
<td>Ubsms</td>
<td>aliyun-python-sdk-ubsms</td>
</tr>
<tr class="even">
<td>Yundun</td>
<td>aliyun-python-sdk-yundun</td>
</tr>
</tbody>
</table>

### Install SDK on no network environment

1. Find an internet accessible computer, access the Python Package Index page https://pypi.python.org.

2. Search SDK package name which listed in the above paragraph “SDK List" and download the file (tar.gz compressed file)

3. Download aliyun-python-sdk-core file (a tar.gz compressed file) from https://pypi.python.org/pypi/aliyun-python-sdk-core/

4. Unzip the aliyun-python-sdk-core file and previously downloaded SDK file.  

5. Copy these unzipped folders to your aliyuncli installed environment.

6. Open your terminal on your aliyuncli installed environment and go to these folders then execute "pip install ."  command. ( aliyun-python-sdk-core at first then other SDK )

### Install Python Environment

`aliyuncli` must run under Python.

If you don’t have Python installed, install version 2.6 or 2.7 using one of the following methods. Version 3 is not supported at this time.

On Windows or OS X, download the Python package for your operating system from python.org and run the installer.

On Linux, OS X, or Unix, install Python using your distribution's package manager.

How to Configure aliyuncli
--------------------------

Before using `aliyuncli` you should create a AccessKey from your console. After login the Aliyun console you can click the like as follows:

&lt;insert method here&gt;

Then you can create the access key and access secret.

Configure the aliyuncli
-----------------------

After creating the access key and access secret, you may configure aliyuncli: :

    $ aliyuncli configure
    Aliyun Access Key ID [None]: <Your aliyun access key id>
    Aliyun Access Key Secret [None]: <Your aliyun access key secret>
    Default Region Id [None]: cn-hangzhou
    Default output format [None]: table

Access key and access secret are certificates invoking the Aliyun open API. Region id is the region area of Aliyun ECS. Output format choices are

-   table
-   JSON
-   text.

Table format sample: :

    <sample>

JSON format sample: :

    <sample>

Text format sample: :

    <sample>


