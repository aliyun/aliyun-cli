#!/usr/bin/env bash

BUCKET="sdk-oss-test"

which aliyun

# sign v4
aliyun oss cat oss://$BUCKET/123.txt --region cn-hangzhou --sign-version v4
aliyun oss ls oss://$BUCKET//123.txt --region cn-hangzhou --sign-version v4

# cleanup
aliyun oss rm oss://$BUCKET/test.txt --region cn-hangzhou

if [ -f ./downloaded.txt ]; then
    rm ./downloaded.txt
fi

# ready to go
echo "version1" > ./test.txt
aliyun oss cp ./test.txt oss://$BUCKET/test.txt --region cn-hangzhou

echo "version2" > ./test.txt
aliyun oss cp ./test.txt oss://$BUCKET/test.txt -f --region cn-hangzhou

aliyun oss cp oss://$BUCKET/test.txt ./downloaded.txt --region cn-hangzhou

OUTPUT=$(cat ./downloaded.txt)

if [[ "$OUTPUT" == "version2" ]];
then
    exit 0;
else
    exit 1;
fi
