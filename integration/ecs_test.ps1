$global:g_var=
$global:g_error=0
$global:force="force"

function do_command() {
    if (-not $($args[1]) -eq $force -and $g_error -eq 1) {
        $global:g_var = 
        return $g_error
    }
    $cmd="$($args[0]) --access-key-id $env:ACCESS_KEY_ID --access-key-secret $env:ACCESS_KEY_SECRET --region cn-hangzhou 2>&1"
    Write-Output "Command<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
    Write-Output $cmd
    Write-Output ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Command"
    $global:g_var=Invoke-Expression $cmd
    Write-Output "Result<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<"
    Write-Output $g_var
    Write-Output ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Result"
    $err= Write-Output $g_var | Select-String -Pattern 'error'
    if (-not $($args[1]) -eq $force -and -not $err -eq ""){
        $global:g_var = 
        $global:g_error=1
        return $g_error
    }
    return 0
}

function ecs_create_instance() {
   do_command "aliyun ecs CreateInstance --ImageId ubuntu_18_04_64_20G_alibase_20190624.vhd --InstanceType ecs.xn4.small" $($args[0])
}

function ecs_start_instance() {
	do_command "aliyun ecs StartInstance --InstanceId $($args[0])" $($args[1])
}

function ecs_stop_instance() {
	do_command "aliyun ecs StopInstance --InstanceId $($args[0])" $($args[1])
}

function ecs_instance_status_wait_until() {
	do_command "aliyun ecs DescribeInstances --InstanceIds `"['$($args[0])']`" --waiter expr=Instances.Instance[0].Status to=$($args[1]) timeout=600" $($args[2])
}

function ecs_describe_instances() {
	do_command "aliyun ecs DescribeInstances" $($args[0])
}

function ecs_delete_instance() {
	do_command "aliyun ecs DeleteInstance --InstanceId $($args[0])" $($args[1])
}

function ecs_get_instance_ids() {
    ecs_describe_instances $force
    $ids=Write-Output $g_var | jq '.Instances.Instance[].InstanceId' -r
    $err=Write-Output $ids | Select-String -Pattern 'error'
    if (-not $err -eq ""){
        $global:g_var=""
    }else {
        $global:g_var=$ids
    }
}

function ecs_clear_all_instances() {
    ecs_get_instance_ids
    $ids=$g_var
   foreach ($id in $ids) {
    Write-Output "###### Try to stop instance $id ######"
    ecs_stop_instance $id $force
   }

   foreach ($id in $ids) {
    Write-Output "###### Try to delete instance $id ######"
    ecs_instance_status_wait_until $id Stopped $force
    ecs_delete_instance $id $force
   }
}

function ecs_test () {

    ecs_create_instance
    $id= Write-Output $g_var | jq '.InstanceId' -r
    Write-Output "###### Try to test instance $id ######"
    ecs_instance_status_wait_until $id Stopped
    ecs_start_instance $id
    ecs_instance_status_wait_until $id Running
    ecs_stop_instance $id $force
    ecs_instance_status_wait_until $id Stopped
    ecs_delete_instance $id
    if ($g_error -eq 1){
        ecs_clear_all_instances
    }
    return $g_error
}


ecs_test

Write-Output "test result is $g_error"

exit $g_error

