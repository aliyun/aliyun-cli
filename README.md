
# 阿里云命令行工具 <Aliyun Command Line Interface>


阿里云命令行工具是用 Python 编写的, 基于阿里云open API 打造的用于管理阿里云资源的统一工具.

通过下载和配置该工具，您可以在同一个命令行方式下控制多个阿里云产品.

阿里云命令行工具代码开源, 并接受开发者的 pull requests. 您可以fork 仓库进行任何修改, 如果遇到问题, 开发团队会直接进行相关支持.

优秀的功能, 我们在审核后, 会吸收进官方的版本中, 并统一发布在阿里云官网.

欢迎通过邮件与我们沟通:

aliyun-developers-efficiency@list.alibaba-inc.com

### 系统要求:

阿里云命令行工具需要系统安装python:

    * 支持2.6.5 及以上版本

### 安装方法:

最简单的安装方式是通过 pip 直接安装, 如果您的系统中有安装 pip 工具, 请执行:

    $ sudo pip install aliyuncli

如果已经安装了阿里云命令行工具, 您可以通过pip升级到最新的版本:

    $ sudo pip install --upgrade aliyuncli

您也可以去阿里云官网下载安装包使用[点击下载](http://market.aliyun.com/products/53690006/cmgj000314.html?spm=5176.900004.4.2.IpMOOc)

windows 版本直接双击MSI 安装即可.
linux 和 MAC os 请执行:

    $ cd <aliyuncli_path>
    $ sudo sh install.sh

### 文件结构:

	aliyuncli/* 是整个的业务逻辑部分, 包含数据的解析, 命令行解析, 以及基本的SDK的调用过程.

	aliyuncli/advance/* 是API聚合逻辑, 这里主要放针对于 aliyuncli 的各种高级功能的开发.

	例如ECS的导入导出功能, RDS的导入导出功能.

	未来会持续的开发更多的高级功能.


### 自动补全功能:
    阿里云命令行工具具备了命令行自动提示和补全的功能. 这个功能安装后不会默认打开, 需要您手动开启:

#### 对于bash:

    $ complete -C '/usr/local/bin/aliyun_completer' aliyuncli

#### 对于zsh:

    % source /usr/local/bin/aliyun_zsh_complete.sh

### 如何使用:

阿里云命令行工具在使用前, 首先需要配置access key 和 access secret, 您可以通过执行下面的命令直接配置:

	$ aliyuncli configure
	Aliyun Access Key ID [****************wQ7v]:
	Aliyun Access Key Secret [****************fxGu]:
	Default Region Id [cn-hangzhou]:
	Default output format [json]:

配置完成后, 您就可以通过执行命令来控制您的云资产:

	$ aliyuncli Ecs DescribeInstances
	$ aliyuncli Ecs StartInstance --InstanceId your_instance_id
	$ aliyuncli Rds DescribeDBInstances
### 如何从源码直接运行:

	$ git clone https://github.com/aliyun/aliyun-cli.git
	$ cd aliyuncli/aliyuncli
	$ python aliyuncli.py ecs DescribeRegions --output json

源码下载后, 可以不安装直接运行, 前提是要安装阿里云python版SDK.

### 通过pip安装阿里云python版SDK
安装阿里云的python版SDK步骤参考如下:

	1. 用curl 或者 浏览器打开"https://bootstrap.pypa.io/get-pip.py" , 将内容保存为pip-install.py
	2. 执行python pip-install.py
	3. pip 安装完毕后, 执行
	pip install aliyun-python-sdk-[productname]

例如, 要安装ECS 产品, 那么就执行:

	$ sudo pip install aliyun-python-sdk-ecs
	
要安装RDS产品SDK, 就执行:

	$ sudo pip install aliyun-python-sdk-rds
	
SLB 则执行:

	$ sudo pip install aliyun-python-sdk-slb

更多介绍, 请参阅官网介绍:

https://help.aliyun.com/document_detail/29993.html
