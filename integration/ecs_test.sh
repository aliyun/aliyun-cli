#!/bin/bash

g_var=
g_error=0
force="force"

do_command() {
	if [[ $2 != $force && $g_error -eq 1 ]]
	then
		g_var=
		return $g_error
	fi

	cmd="$1 --access-key-id $ACCESS_KEY_ID --access-key-secret $ACCESS_KEY_SECRET --region $REGION_ID 2>&1"

	echo "Command<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	echo $cmd
	echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Command"

	g_var=$(eval $cmd)

	echo "Result<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
	echo $g_var
	echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Result"

	err=$(echo $g_var | grep -i -e "error" -e "panic")

	if [[ $2 != $force && $err != "" ]]
	then
		g_var=
		g_error=1
		return $g_error
	fi

	return 0
}

ecs_create_instance() {
	do_command "aliyun ecs CreateInstance --ImageId 'aliyun_2_1903_x64_20G_alibase_20231221.vhd' --InstanceType 'ecs.t5-lc1m1.small' --InstanceChargeType PostPaid" $1
}

ecs_start_instance() {
	do_command "aliyun ecs StartInstance --InstanceId $1" $2
}

ecs_stop_instance() {
	do_command "aliyun ecs StopInstance --InstanceId $1" $2
}

ecs_instance_status_wait_until() {
	do_command "aliyun ecs DescribeInstances --InstanceIds ['$1'] --waiter expr=Instances.Instance[0].Status to=$2 timeout=600" $3
}

ecs_describe_instances() {
	do_command "aliyun ecs DescribeInstances" $1
}

ecs_delete_instance() {
	do_command "aliyun ecs DeleteInstance --InstanceId $1" $2
}

ecs_get_instance_ids() {
	ecs_describe_instances $force
	ids=$(echo $g_var | jq '.Instances.Instance[].InstanceId')
	err=$(echo ids | grep -i "error")
	if [[ $err != "" ]]
	then
		g_var=""
	else
		g_var=$ids
	fi
}

ecs_clear_all_instances() {
	ecs_get_instance_ids

	ids=$g_var

	for id in ${ids[@]}; do
		echo "###### Try to stop instance $id ######"
		ecs_stop_instance $id $force
	done

	for id in ${ids[@]}; do
		echo "###### Try to delete instance $id ######"

		ecs_instance_status_wait_until $id Stopped $force

		ecs_delete_instance $id $force
	done
}

ecs_test() {
	ecs_create_instance

	id=$(echo $g_var | jq '.InstanceId')

	echo "###### Try to test instance $id ######"

	ecs_instance_status_wait_until $id Stopped

	ecs_start_instance $id

	ecs_instance_status_wait_until $id Running

	ecs_stop_instance $id $force

	ecs_instance_status_wait_until $id Stopped

	ecs_delete_instance $id

	if [[ $g_error -eq 1 ]]
	then
		ecs_clear_all_instances
	fi

	return $g_error
}

ecs_test

echo "test result is $g_error"

exit $g_error
