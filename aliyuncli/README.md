aliyuncli

This tool is based on aliyun open api sdk and provides a unified command line interface to Aliyun Web Service.
You can use this tool to manage your resource of aliyun, such ad ECS, RDS, YunDun, Mts and so on.

The aliyuncli package works on Python versions:

2.6.5 and greater

Installation:

The easiest way to install aliyuncli is to use pip tool:

    $ pip install aliyuncli

or you should use:

    $ sudo pip install aliyuncli

If you have installed teh aliyuncli , you can also upgrade to the latest version :

    $ pip install --upgrade aliyuncli

Another method to install aliyuncli is downloading the package first and then go into the
directory and run install script:

    $ cd <patch_to_aliyuncli>
    $ sh install.sh

Command complete:

The tool support command auto complete, after you install the package successful ,you can find the aliyun_completer
first and then use complete inline command of bash to enable the command auto complete.

    $ complete -C 'patch_to_aliyun_completer' aliyuncli

If you want to keep auto complete when you close the shell, you should write it in .bash_profile. You can just only
write the above command to the file. When you restart the shell , the command will take effect automatically.

Document:

The document you can check http://www.aliyun.com to find all document for aliyun command line interface.





