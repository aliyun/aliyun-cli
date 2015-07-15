#!/bin/sh
export PYTHONPATH=`pwd`
cd ./packages

InstallPython()
{
    tar -zxvf Python-2.7.tgz
    cd Python-2.7 && sh ./configure && make
    sudo make install
    cd .. 
    sudo rm -rf Python-2.7
}


InstallSetupTool()
{
    tar -zvxf setuptools-2.0.1.tar.gz
    cd setuptools-2.0.1
    sudo python setup.py install
    cd ..
    sudo rm -rf setuptools-2.0.1
}
InstallPip()
{
    tar -zvxf pip-1.3.1.tar.gz 
    cd pip-1.3.1 
    sudo python setup.py install
    cd ..
    sudo rm -rf  pip-1.3.1
}
InstallAliyuncli()
{
    tar -zvxf aliyuncli-1.0.0.tar.gz
    cd aliyuncli-1.0.0
    sudo python setup.py install --record ../../files.txt
    cd ..
    sudo rm -rf aliyuncli-1.0.0
}
pipv=`which pip 2>&1| awk -F: '{print $1}' | awk -F'/' '{print $NF}'`
if [ "$pipv" = "pip" ]
    then
        echo 'Pip already installed'
    else
        InstallPip
fi
pyv=`which python 2>&1| awk -F: '{print $1}' | awk -F'/' '{print $NF}'`
if [ "$pyv" = "python" ]
    then
        echo 'Python already installed'
    else
        InstallPython
fi 

aliyuncli=`which aliyuncli 2>&1| awk -F: '{print $1}' | awk -F'/' '{print $NF}'`
if [ "$aliyuncli" = "aliyuncli" ]
then
    echo "Aliyuncli has been installed !!!"
    echo -e "Do you want to remove the exisitng version and install the current version ? [Y/N]\c "
    read Choice
    
    case $Choice in
    y|Y|[Yy][Ee][Ss])
        InstallAliyuncli
        echo  ""
	echo "***********************************************************"
        echo  "* New aliyuncli version has been installed successfully ! *"
	echo "***********************************************************"
        ;;
    *)
        echo  "Skip the installation of this version!"
        ;;
    esac

else
    InstallSetupTool
    InstallAliyuncli
    echo  ""
    echo "***********************************************************"
    echo  "* New aliyuncli version has been installed successfully ! *"
    echo "***********************************************************"
fi
complete -C '/usr/local/bin/aliyun_completer' aliyuncli
