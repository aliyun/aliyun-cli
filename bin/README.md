## Pre-requisite

1. Make sure aliyun binary is installed on your laptop and add location of aliyun into system env variable PATH
2. Install jq on your laptop thru 'brew install jq`./


## Install convient script to your laptop

1. `clone git@gitlab.alibaba-inc.com:aliopensource/aliyun-cli.git ` then `cd aliyun-cli`
2. Add <aliyun-cli repo path>/bin into PATH of env variable `echo PAHT=<aliyun-cli repo path>/bin:%PATH >> ~/.zshrc`
3. Add following alias to your ~/.zshrc or ~/.bashrc

```
alias aliyun-cn-qingdao="print_table cn-qingdao "
alias aliyun-cn-beijing="print_table cn-beijing "
alias aliyun-cn-zhangjiakou="print_table cn-zhangjiakou "
alias aliyun-cn-huhehaote="print_table cn-huhehaote "
alias aliyun-cn-hangzhou="print_table cn-hangzhou "
alias aliyun-cn-shanghai="print_table cn-shanghai "
alias aliyun-cn-shenzhen="print_table cn-shenzhen "
alias aliyun-cn-hongkong="print_table cn-hongkong "
alias aliyun-ap-northeast-1="print_table ap-northeast-1 "
alias aliyun-ap-southeast-1="print_table ap-southeast-1 "
alias aliyun-ap-southeast-2="print_table ap-southeast-2 "
alias aliyun-ap-southeast-3="print_table ap-southeast-3 "
alias aliyun-us-east-1="print_table us-east-1 "
alias aliyun-us-west-1="print_table us-west-1 "
alias aliyun-me-east-1="print_table me-east-1 "
alias aliyun-eu-central-1="print_table eu-central-1 "
alias aliyun-region="aliyun ecs DescribeRegions  | jq -r .Regions.Region | jq -r '.[] | .RegionId'"
```

4. 'source ~/.zshrc' to activate setting in shell env.

#### Have a summary of ecs in different region. 

Type aliyun-cn + Tab to select any region to view list of ecs.

```
aliyun-cn-hangzhou
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status
---------------------	---------------------	-------------	-------------	-------------	------
i-bp19uitdpeqcerstqk3a	iZbp19uitdpeqcerstqk3aZ		192.168.33.74	cn-hangzhou-f	Running
i-bp17esjfzl57qv1ym05n	iZbp17esjfzl57qv1ym05nZ	116.62.163.58	192.168.33.73	cn-hangzhou-f	Running
i-bp185dy2o3o6lnlo4f07	iZbp185dy2o3o6lnlo4f07Z	116.62.223.146	192.168.33.71	cn-hangzhou-f	Running
i-bp185dy2o3o6lnlo4f06	iZbp185dy2o3o6lnlo4f06Z	47.96.15.179	192.168.33.72	cn-hangzhou-f	Running
i-bp185dy2o3o6lnlo4f05	iZbp185dy2o3o6lnlo4f05Z	47.96.16.255	192.168.33.70	cn-hangzhou-f	Running
i-bp1e7obsg3ccc6fgyil4	iZbp1e7obsg3ccc6fgyil4Z	116.62.16 8.131	192.168.33.69	cn-hangzhou-f	Running
i-bp1e7obsg3ccc6fgyil5	iZbp1e7obsg3ccc6fgyil5Z	118.31.64.70	192.168.33.68	cn-hangzhou-f	Running
i-bp1e7obsg3ccc6fgyil3	iZbp1e7obsg3ccc6fgyil3Z	47.96.5.184	192.168.33.67	cn-hangzhou-f	Running
i-bp1fveanl3zy002cfyyv	iZbp1fveanl3zy002cfyyvZ	118.31.65.19	192.168.33.66	cn-hangzhou-f	Running
i-bp1fveanl3zy002cfyyw	iZbp1fveanl3zy002cfyywZ	47.96.14.140	192.168.33.59	cn-hangzhou-f	Running
```


#### Lookup any ECS instance from any region with instanceId, private ip, public ip or hostname

`aliyun-lookup <any instanceid, public/private ip, hostname>`


```
#aliyun-lookup 47.75.14.57

ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
```

#### Reload system disk with original image

`aliyun-reload <any instanceid, public/private ip, hostname>`


```
aliyun-reload i-j6cg0s7swgbn77gmrcll

ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1		10.1.2.48	cn-hongkong-b	Stopped	centos_7_04_64_20G_alibase_201701015.vhd
Stopping i-j6cg0s7swgbn77gmrcll at cn-hongkong, current status: Stopped
Replacing system of i-j6cg0s7swgbn77gmrcll at cn-hongkong, current status: Stopped
{"RequestId":"6308CFF1-6DB6-46B9-9125-EE12DCF0AF7F","DiskId":"d-j6c4pvlm72odu56kolgx"}
Starting system of i-j6cg0s7swgbn77gmrcll at cn-hongkong, current status: Stopped
{"RequestId":"46FB6486-DB92-455F-B5C3-4FEE18FBF805"}
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Running	centos_7_04_64_20G_alibase_201701015.vhd
```

#### Reload instance with specific system disk

`aliyun-reload <any instanceid, public/private ip, hostname> <image_id>`

e.g.

`aliyun-reload i-bp185dy2o3o6lnlo4f07 sles_12_sp2_64_20G_alibase_20170907.vhd`


#### Stop any instance with instanceId, private ip, public ip or hostname

`aliyun-stop i-j6cg0s7swgbn77gmrcll`

```
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1		10.1.2.48	cn-hongkong-b	Stopped	centos_7_04_64_20G_alibase_201701015.vhd
```

#### Force to stop an instance or reload

```
#export FORCE=true
#aliyun-stop 192.168.33.71

ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-bp185dy2o3o6lnlo4f07	iZbp185dy2o3o6lnlo4f07Z		192.168.33.71	cn-hangzhou-f	Stopped	sles_12_sp2_64_20G_alibase_20170907.vhd
```

#### Start any instance with instanceId, private ip, public ip or hostname

`aliyun-start <any instanceid, public/private ip, hostname>`

```
#aliyun-start i-j6cg0s7swgbn77gmrcll

ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1		10.1.2.48	cn-hongkong-b	Stopped	centos_7_04_64_20G_alibase_201701015.vhd
{"RequestId":"3F62E983-1C46-4425-A782-86A31ABA997A"}
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Starting	centos_7_04_64_20G_alibase_201701015.vhd
```

#### Reboot ECS with instanceId, private ip, public ip or hostname

`aliyun-reboot <any instanceid, public/private ip, hostname>`

```
#aliyun-reboot 47.75.14.57

ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Running	centos_7_04_64_20G_alibase_201701015.vhd
{"RequestId":"44EE2819-66F4-46E9-AC66-1447A91D2D2C"}
ID                	Hostname            	Public IP	Internal IP	ZoneId	    Status	ImageId
---------------------	---------------------	-------------	-------------	-------------	------	------------------
i-j6cg0s7swgbn77gmrcll	eric-hongkong-1	47.75.14.57	10.1.2.48	cn-hongkong-b	Stopping	centos_7_04_64_20G_alibase_201701015.vhd

```



