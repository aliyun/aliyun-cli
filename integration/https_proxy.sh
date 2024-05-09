#!/bin/bash

export HTTPS_PROXY=https://1.2.3.4:8080/

cmd="aliyun sts GetCallerIdentity --access-key-id $ACCESS_KEY_ID --access-key-secret $ACCESS_KEY_SECRET --region $REGION_ID  2>&1"
g_var=$(eval $cmd)

err=$(echo $g_var | grep -i -e "proxyconnect tcp" -e "timeout")

if [[ $err == "" ]]
then
    exit 1
fi

exit 0
