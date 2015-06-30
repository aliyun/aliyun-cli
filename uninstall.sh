#!/bin/sh
export PYTHONPATH=`pwd`
UnInstallAliyuncli()
{
    if [ ! -f "files.txt" ]
    then
        aliyunclipath=`which aliyuncli`
        sudo rm -f $aliyunclipath
    else
        cat files.txt | xargs sudo rm -rf 
        sudo rm -f files.txt 
    fi
}
aliyuncli=`which aliyuncli 2>&1| awk -F: '{print $1}' | awk -F'/' '{print $NF}'`
if [ "$aliyuncli" = "aliyuncli" ]
then
    echo "aliyuncli is to be uninstalled"
    UnInstallAliyuncli
else
    echo "aliyuncli have not  been installed"
fi
