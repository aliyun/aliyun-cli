Aliyun Command Line Interface
=============================
Overview
------------------
Aliyun Command Line Interface ``aliyuncli`` is a unified tool to manage your Aliyun services. From this tool you can easily invoke the Aliyun open API. Also, you can control multiple Aliyun services from the command line and automate them through scripts, like the Bash shell or Python. 

Aliyuncli on Github
----------------------
The ``aliyuncli`` has been uploaded to Github and anyone can fork the code freely. You can access it at: https://github.com/aliyun/aliyun-cli

How to install aliyuncli
^^^^^^^^^^^^^^^^^^^^^^^^
Aliyun provides two way to install the aliyuncli:

1. install aliyuncli using pip
2. install from software package

Install aliyuncli using pip
^^^^^^^^^^^^^^^^^^^^^^^^^^^
If pip is installed in your operating system, whether Windows, Linux, or Mac OS, you can install ``aliyuncli`` using pip:

Windows
^^^^^^^
::

 pip install aliyuncli

To upgrade the existing aliyuncli , use the ``--upgrade`` option:
::	

 pip install --upgrade aliyuncli

Linux, Mac OS and Unix
^^^^^^^^^^^^^^^^^^^^^^
::

 $ sudo pip install aliyuncli

To upgrade the existing aliyuncli , use the ``--upgrade`` option:
::

 $ pip install --upgrade aliyuncli


Install from software package
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

If you don't have the pip tool, you can also install ``aliyuncli`` from a Aliyun supplied software package.

Aliyuncli supports several operating systems:

* Windows
* Linux
* Mac OS. 

You can choose the method for the different OS.

You can find the software package at the following link http://market.aliyun.com/products/53690006/cmgj000314.html?spm=5176.900004.4.2.esAaC2

It is a free download. 

The package contains three install packages: 

* ``cli.tar.gz`` is for Linux and mac OS 
* ``AliyunCLI_x86`` is for Windows 32 bit OS 
* ``AliyunCLI_x64`` is for Windows 64 bit OS

Windows
^^^^^^^^^^^^^^^^

1. Find ``AliyunCLI.msi`` and double click the msi, you will go into the installation guide.
2. Click the “next” button and choose your desired path and confirm
3. Finish the install

Linux and Mac OS
^^^^^^^^^^^^^^^^^^^^^^^^^

Install as follows:
::

 $ tar -zxvf cli.tar.gz
 $ cd cli
 $ sudo sh install.sh

Check the aliyuncli Installation
--------------------------------

Confirm the aliyuncli installed correctly by viewing the help file:
::

	$ aliyuncli help

or 

::

	$ aliyuncli

How to install Aliyun Python SDK
-----------------------------------

``aliyuncli`` must work with Aliyun Python sdk 2.0 You should install the SDK after you install ``aliyuncli``, otherwise you can not access the Aliyun service.


Install SDK using pip
^^^^^^^^^^^^^^^^^^^^^^^^^^
Aliyun Python SDK can only be installed by pip. Since each product of aliyun has their own SDK, 
you can install a required SDK individually with no need install all of them.

For example, if you need only the ECS SDK, you can install only it as follows:
::

 $ sudo pip install aliyun-python-sdk-ecs

If you need only the RDS SDK:
::

 $ sudo pip install aliyun-python-sdk-rds

For SLB:
::

 $ sudo pip install aliyun-python-sdk-slb

The SDK list
^^^^^^^^^^^^

Product|SDK
----|----
BatchCompute	|aliyun-python-sdk-batchcompute
Bsn				|aliyun-python-sdk-bsn
Bss				|aliyun-python-sdk-bss
Cms				|aliyun-python-sdk-cms
Crm				|aliyun-python-sdk-crm
Drds			|aliyun-python-sdk-drds
Ecs				|aliyun-python-sdk-ecs
Ess				|aliyun-python-sdk-ess
Ft				|aliyun-python-sdk-ft
Ocs				|aliyun-python-sdk-ocs
Oms				|aliyun-python-sdk-oms
OssAdmin		|aliyun-python-sdk-ossadmin
Ram				|aliyun-python-sdk-ram
Rds				|aliyun-python-sdk-rds
Risk			|aliyun-python-sdk-risk
R-kvstore		|aliyun-python-sdk-r-kvstore
Slb				|aliyun-python-sdk-slb
Sts				|aliyun-python-sdk-sts
Ubsms			|aliyun-python-sdk-ubsms
Yundun			|aliyun-python-sdk-yundun



Install Python Environment
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Aliyuncli must run under python environment, so please make sure your operation system has installed python environment. 

If you don’t have python installed , installed version 2.6 or 2.7 (not support 3.X now) using one of the following methods:


On Windows or OS X, download the Python package for your operating system from python.org and run the installer.

On Linux, OS X, or Unix, install Python using your distribution's package manager.

How to Configure aliyuncli
-----------------------------
Before using aliyuncli you should create a AccessKey from your console. After login the aliyun console you can click the like as follow: 

Then you can create the access key and access secret:

Configure the aliyuncli quickly
----------------------------------

After create access key and access secret , you can configure aliyuncli quickly:

	$ aliyuncli configure
	Aliyun Access Key ID [None]: <Your aliyun access key id>
	Aliyun Access Key Secret [None]: <Your aliyun access key secret>
	Default Region Id [None]: cn-hangzhou
	Default output format [None]: table

Access key and access secret are certificate invoke the aliyun open API. Region id is the region area of aliyun ECS. Output format you can choose is table , json and text.

Table format likes:
 
Json format likes:
 
Text format like:

 
You can choose one format as your wish. 


How to use aliyuncli
-----------------------

aliyuncli has four parts:


First part is the name of the tool “aliyuncli”

Second part is the available service name, such as: ecs , rds, slb, ots

The third part is the available operation of each service.

The final part is the list of keys and values, this part can has multiple keys and values. The values can be number, string or json format. 

Here are some examples:

	$ aliyuncli rds DescribeDBInstances --PageSize 50
	$ aliyuncli ecs DescribeRegions
	$ aliyuncli rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx

More usage
^^^^^^^^^^^^^^

	--filter
Aliyuncli supports filter function. When we call any open API , the data from the server is json format by default. And filter function can help user handle the "json" format data easily. 

Here are some examples:
::

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

Command Completion
---------------------

On Unix-like systems, the aliyuncli includes a command-completion feature that enables you to use the TAB key to complete a partially typed command. This feature is not automatically installed so you need to configure it manually.


Configuring command completion requires two pieces of information: the name of the shell you are using and the location of aliyun_completer script.

Check your shell:
^^^^^^^^^^^^^^^^^^^^^

Current aliyuncli only supports two shells: bash and zsh. 

1. find aliyun_completer, you can use:
::

	$ which aliyun_completer
	/usr/local/bin/aliyun_completer

2. enable command completion:

bash - use the build-in command complete:


	$ complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli
zsh - source bin/aliyun_zsh_completer.sh

	% source /usr/local/bin/aliyun_zsh_completer.sh
	
Test Command Completion
^^^^^^^^^^^^^^^^^^^^^^^^^^^
::

	$ aliyuncli sTAB
	ecs     rds     slb
The services showing dependences the sdk you installed. 

Finally, to ensure that completion continues to work after a reboot, add the configuration command that you used to enable command completion to your shell profile.
::

	$ vim ~/.bash_profile
	
Add ``complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli`` at the end.
