Aliyun Command Line Interface
=============================
Overview
------------------
Aliyun Command Line Interface ``aliyuncli`` is a unified tool to manage your Aliyun services. Using this tool you can easily invoke the Aliyun open API to control multiple Aliyun services from the command line and also automate them through scripts, for instance using the Bash shell or Python. 

Aliyuncli on Github
----------------------
The ``aliyuncli`` tool is on Github and anyone can fork the code, subject to the license. You can access it at: https://github.com/aliyun/aliyun-cli

How to Install aliyuncli
^^^^^^^^^^^^^^^^^^^^^^^^
Aliyun provides two ways to install the ``aliyuncli`` tool:

1. Install using pip
2. Install from a software package

Install aliyuncli Using pip
^^^^^^^^^^^^^^^^^^^^^^^^^^^
If you have Windows, Linux, or Mac OS and pip is installed in your operating system, 
you can install ``aliyuncli`` using pip:

Windows
^^^^^^^
::

 pip install aliyuncli

To upgrade the existing ``aliyuncli``, use the ``--upgrade`` option:
::	

 pip install --upgrade aliyuncli

Linux, Mac OS and Unix
^^^^^^^^^^^^^^^^^^^^^^
::

 $ sudo pip install aliyuncli

To upgrade the existing ``aliyuncli``, use the ``--upgrade`` option:
::

 $ pip install --upgrade aliyuncli

Install from a Software Package
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

If you don't have the pip tool, you can also install ``aliyuncli`` from an Aliyun supplied software package.

Aliyuncli supports several operating systems with the package:

* Windows
* Linux
* Mac OS.

You can find the software package for a free download at the following link http://market.aliyun.com/products/53690006/cmgj000314.html?spm=5176.900004.4.2.esAaC2

The package contains three install packages: 

* ``cli.tar.gz`` is for Linux and Mac OS 
* ``AliyunCLI_x86`` is for Windows 32 bit OS 
* ``AliyunCLI_x64`` is for Windows 64 bit OS

Windows
^^^^^^^

1. Find ``AliyunCLI.msi`` and double click the msi. You will go into the installation guide.
2. Click the “next” button and choose your desired path and confirm
3. Finish the install

Linux and Mac OS
^^^^^^^^^^^^^^^^

Install as follows:
::

 $ tar -zxvf cli.tar.gz
 $ cd cli
 $ sudo sh install.sh

Check the aliyuncli Installation
--------------------------------

Confirm that ``aliyuncli`` installed correctly by viewing the help file:
::

	$ aliyuncli help

or 

::

	$ aliyuncli

How to Install the Aliyun Python SDK
------------------------------------

``aliyuncli`` requires the Aliyun Python SDK 2.0. 
You should install the SDK after you install ``aliyuncli``, otherwise you can not access the Aliyun service.

Install SDK Using pip
^^^^^^^^^^^^^^^^^^^^^
The Aliyun Python SDK can only be installed by pip. 

Since each Aliyun service has their own SDK, 
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

SDK List
^^^^^^^^

+--------------+--------------------------------+
| Product      |  SDK                           |
+==============+================================+
| BatchCompute | aliyun-python-sdk-batchcompute |
+--------------+--------------------------------+
| Bsn          | aliyun-python-sdk-bsn          |
+--------------+--------------------------------+
| Bss          | aliyun-python-sdk-bss          |
+--------------+--------------------------------+
| Cms          | aliyun-python-sdk-cms          |
+--------------+--------------------------------+
| Crm          | aliyun-python-sdk-crm          |
+--------------+--------------------------------+
| Drds         | aliyun-python-sdk-drds         |
+--------------+--------------------------------+
| Ecs          | aliyun-python-sdk-ecs          |
+--------------+--------------------------------+
| Ess          | aliyun-python-sdk-ess          |
+--------------+--------------------------------+
| Ft           | aliyun-python-sdk-ft           |
+--------------+--------------------------------+
| Ocs          | aliyun-python-sdk-ocs          |
+--------------+--------------------------------+
| Oms          | aliyun-python-sdk-oms          |
+--------------+--------------------------------+
| OssAdmin     | aliyun-python-sdk-ossadmin     |
+--------------+--------------------------------+
| Ram          | aliyun-python-sdk-ram          |
+--------------+--------------------------------+
| Ocs          | aliyun-python-sdk-ocs          |
+--------------+--------------------------------+
| Rds          | aliyun-python-sdk-rds          |
+--------------+--------------------------------+
| Risk         | aliyun-python-sdk-risk         |
+--------------+--------------------------------+
| R-kvstore    | aliyun-python-r-kvstore        |
+--------------+--------------------------------+
| Slb          | aliyun-python-sdk-slb          |
+--------------+--------------------------------+
| Ubsms        | aliyun-python-sdk-ubsms        |
+--------------+--------------------------------+
| Yundun       | aliyun-python-sdk-yundun       |
+--------------+--------------------------------+

Install Python Environment
^^^^^^^^^^^^^^^^^^^^^^^^^^

``aliyuncli`` must run under Python. 

If you don’t have Python installed, install version 2.6 or 2.7 using one of the following methods. Version 3 is not supported at this time.

On Windows or OS X, download the Python package for your operating system from python.org and run the installer.

On Linux, OS X, or Unix, install Python using your distribution's package manager.

How to Configure aliyuncli
-----------------------------
Before using ``aliyuncli`` you should create a AccessKey from your console. After login the Aliyun console you can click the like as follows: 

<insert method here>

Then you can create the access key and access secret.

Configure the aliyuncli
-----------------------

After creating the access key and access secret, you may configure aliyuncli:
::

	$ aliyuncli configure
	Aliyun Access Key ID [None]: <Your aliyun access key id>
	Aliyun Access Key Secret [None]: <Your aliyun access key secret>
	Default Region Id [None]: cn-hangzhou
	Default output format [None]: table

Access key and access secret are certificates invoking the Aliyun open API. 
Region id is the region area of Aliyun ECS. 
Output format choices are 

* table
* JSON
* text.

Table format sample:
::

 <sample>

JSON format sample:
::
 
 <sample>
 
Text format sample:
::

 <sample>

How to Use aliyuncli
-----------------------

An ``aliyuncli`` command has four parts:

* Name of the tool “aliyuncli”
* Service name, such as: ecs, rds, slb, ots
* Available operations for each service
* List of keys and values, with possible multiple keys and values. The values can be number, string, or JSON format. 

Here are some examples:
::

 $ aliyuncli rds DescribeDBInstances --PageSize 50
 $ aliyuncli ecs DescribeRegions
 $ aliyuncli rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx

Additional Usage Information
^^^^^^^^^^^^^^^^^^^^^^^^^^^^
::

 --filter

``aliyuncli`` supports a filter function. When any API is called, the data returned is JSON formatted by default. 
The filter function can help the user manipulate the JSON formatted data more easily. 

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

On Unix-like systems, the ``aliyuncli`` includes a command-completion feature 
that enables you to use the ``TAB`` key to complete a partially typed command. 
This feature is not automatically installed, so you need to configure it manually.

Configuring command completion requires two pieces of information:

* the name of the shell you are using
* the location of ``aliyun_completer`` script.

Check Your Shell
^^^^^^^^^^^^^^^^

Currently ``aliyuncli`` supports these shells: 

* bash
* zsh. 

1. To find the ``aliyun_completer``, you can use:
::

 $ which aliyun_completer
 /usr/local/bin/aliyun_completer

2. To enable command completion:

bash - use the build-in command complete:
::

 $ complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli
	
zsh - source bin/aliyun_zsh_completer.sh
::

 % source /usr/local/bin/aliyun_zsh_completer.sh
	
Test Command Completion
^^^^^^^^^^^^^^^^^^^^^^^^^^^
::

	$ aliyuncli s<TAB>
	ecs     rds     slb

The services display the SDK(s) you installed. 

Finally, to ensure that completion continues to work after a reboot, 
add a configuration command to enable command completion to your shell profile.
::

	$ vim ~/.bash_profile
	
Add ``complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli`` at the end of the file.
