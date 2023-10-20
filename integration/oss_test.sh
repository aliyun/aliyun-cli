#!/usr/bin/env bash

BUCKET="sdk-oss-test"

# cleanup
aliyun oss rm oss://$BUCKET/test.txt
rm ./downloaded.txt

# ready to go
echo "version1" > ./test.txt
aliyun oss cp ./test.txt oss://$BUCKET/test.txt

echo "version2" > ./test.txt
aliyun oss cp ./test.txt oss://$BUCKET/test.txt -f

aliyun oss cp oss://$BUCKET/test.txt ./downloaded.txt

OUTPUT=$(cat ./downloaded.txt)

if [[ "$OUTPUT" == "version2" ]];
then
    exit 0;
else
    exit 1;
fi
