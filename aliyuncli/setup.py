#!/usr/bin/env python
'''
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing,
 software distributed under the License is distributed on an
 "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 KIND, either express or implied.  See the License for the
 specific language governing permissions and limitations
 under the License.
'''

import os, sys
from setuptools import setup, Command,find_packages


#long_description = open("README.rst").read()
def main():
    setup(
        name='aliyuncli',
        description='Universal Command Line Environment for aliyun',
        version='0.0.1',
        url='http://www.aliyun.com/',
        packages = find_packages(),
        platforms=['unix', 'linux'],
        author='xxx',
        author_email='xxx',
        entry_points = {
            'console_scripts': [
                'aliyuncli = aliyuncli.aliyuncli:main',
                'aliyun_completer  = aliyuncli.aliyun_completer:aliyun_complete',

            ]
        }
        # the following should be enabled for release
    )


if __name__ == '__main__':
    main()
