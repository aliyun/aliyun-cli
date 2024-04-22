#!/bin/bash

g_var=
g_error=0

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

vpc_test() {
	which aliyun

	do_command "aliyun vpc CreateVpc"

	id=$(echo $g_var | jq '.VpcId')

	echo "###### Try to test instance $id ######"

	sleep 3

	do_command "aliyun vpc DeleteVpc --VpcId $id"

	return $g_error
}

vpc_test

echo "test result is $g_error"

exit $g_error
