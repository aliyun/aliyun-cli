##Aliyun Command Line Interface
###Brief introduction
Aliyun Command Line Interface (aliyuncli) is a unified tool to manage your Aliyun services. From this tool you can invoke the open API of Aliyun productions easily. With just this one tool to download and configure , you can control multiple Aliyun services from the command line and automate them through scripts (like shell or python). 


###Aliyuncli on github

Now aliyuncli has been uploaded the source code to github, and any one can fork the code freely. You can access the url of github: https://github.com/aliyun/aliyun-cli




###How to install aliyuncli

Now aliyun provides two way to install the aliyuncli:

1. install aliyuncli using pip

2. install from software package


####Install aliyuncli using pip:
If you have installed the pip in your operation system, no matter windows Linux or Mac OS, you can install aliyuncli using pip:
####Windows:
	

	> pip install aliyuncli

To upgrade the existing aliyuncli , use the --upgrade option:
	

	> pip install --upgrade aliyuncli

####Linux , Mac OS and Unix:

	$ sudo pip install aliyuncli

To upgrade the existing aliyuncli , use the --upgrade option:

	$ pip install --upgrade aliyuncli


###Install from software package

If you dont have the pip tool, you can also install aliyuncli from software package.

Aliyuncli supports all kind of operation systems: Windows , Linux and mac OS. You can choose the method by different OS.

You can find the software package from the following link:
	
[click and download](http://market.aliyun.com/products/53690006/cmgj000314.html?spm=5176.900004.4.2.esAaC2)

It is free downloading now. The package contains three install packages: 
cli.tar.gz is for Linux and mac OS, AliyunCLI_x86 is for windows 32 OS and AliyunCLI_x64 is for windows 64 OS. 

You can choose the install package according your operation system.


####For windows:
1.	Find AliyunCLI.msi and double click the msi, you will go into the installation guide.

2.	Click the “next” button and choose your favorite path and confirm


3.	finish the install



 

####For Linux and Mac OS:
You can install like follow step:

	$ tar -zxvf cli.tar.gz
	$ cd cli
	$ sudo sh install.sh



###Check the aliyuncli installation:


Confirm the aliyuncli installed correctly by viewing the help file:

	$ aliyuncli help

or 
dddd
	$ aliyuncli

###How to install Aliyun python SDK
aliyuncli must work with aliyun python sdk(2.0) , you should install the sdk after you install the aliyuncli. Otherwise you can not access the aliyun service normally.


####Install SDK using pip:
Aliyun python sdk only can be installed by pip. So please make sure your operation system has installed pip. Each product of aliyun has one sdk , you can install the required sdk one by one and no need install all of them.


Such as you need ECS sdk, you just install it as following command:

	$ sudo pip install aliyun-python-sdk-ecs
If you need RDS sdk, you just install it using:

	$ sudo pip install aliyun-python-sdk-rds
For SLB, you using:

	$ sudo pip install aliyun-python-sdk-slb


####The SDK list:

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



	

####Install python environment:


Aliyuncli must run under python environment, so please make sure your operation system has installed python environment. 

If you don’t have python installed , installed version 2.6 or 2.7 (not support 3.X now) using one of the following methods:


On Windows or OS X, download the Python package for your operating system from python.org and run the installer.

On Linux, OS X, or Unix, install Python using your distribution's package manager.



###How to configure aliyuncli
Before using aliyuncli you should create a AccessKey from your console. After login the aliyun console you can click the like as follow: 


Then you can create the access key and access secret:



###Configure the aliyuncli quickly

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


###How to use aliyuncli
aliyuncli has four parts:


First part is the name of the tool “aliyuncli”

Second part is the available service name, such as: ecs , rds, slb, ots

The third part is the available operation of each service.

The final part is the list of keys and values, this part can has multiple keys and values. The values can be number, string or json format. 

Here are some examples:

	$ aliyuncli rds DescribeDBInstances --PageSize 50
	$ aliyuncli ecs DescribeRegions
	$ aliyuncli rds DescribeDBInstanceAttribute --DBInstanceId xxxxxx

####More usage
	--filter
Aliyuncli supports filter function. When we call any open API , the data from the server is json format by default. And filter function can help user handle the "json" format data easily. 

Here are some examples:

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




###Command Completion
On Unix-like systems, the aliyuncli includes a command-completion feature that enables you to use the TAB key to complete a partially typed command. This feature is not automatically installed so you need to configure it manually.


Configuring command completion requires two pieces of information: the name of the shell you are using and the location of aliyun_completer script.
####Check your shell:
Current aliyuncli only supports two shells: bash and zsh. 


1.find aliyun_completer, you can use:

	$ which aliyun_completer
	/usr/local/bin/aliyun_completer
2.enable command completion:


bash - use the build-in command complete:


	$ complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli
zsh - source bin/aliyun_zsh_completer.sh

	% source /usr/local/bin/aliyun_zsh_completer.sh
####Test Command Completion

	$ aliyuncli sTAB
	ecs     rds     slb
The services showing dependences the sdk you installed. 

Finally, to ensure that completion continues to work after a reboot, add the configuration command that you used to enable command completion to your shell profile.


	$ vim ~/.bash_profile
	Add complete -C ‘/usr/local/bin/aliyun_completer’ aliyuncli to the end line.
