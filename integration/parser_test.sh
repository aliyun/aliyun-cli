#!/usr/bin/env bash

echo "configure list"
go run main/main.go configure list

echo "oss ls"
go run main/main.go oss ls

echo "oss ls --region cn-hangzhou"
go run main/main.go oss ls --region cn-hangzhou

echo "ls oss://weeping/tingwu --region cn-shanghai --sign-version v4"
go run main/main.go oss ls oss://weeping/tingwu --region cn-shanghai --sign-version v4

echo "sts GetCallerIdentity"
go run main/main.go sts GetCallerIdentity

echo "sts GetCallerIdentity --region cn-beijing"
go run main/main.go sts GetCallerIdentity --region cn-beijing
