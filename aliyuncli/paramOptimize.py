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
#/usr/bin/env python
#!-*- coding:utf-8 -*-
import os
import urllib2
import cliError


def _paramOptimize(keyValues):
    if keyValues is None:
        return
    if not isinstance(keyValues,dict):
        return
    for key in keyValues.keys():
        if keyValues[key] is not None and len(keyValues[key]) >0:
            _getIndirectValues(keyValues[key])
        else:
            continue

def _getIndirectValues(values_list):
    if values_list is None:
        return
    if not isinstance(values_list,list):
        return
    count = len(values_list)
    for index in range(0,count):
        for prefix in PrefixMap:
            if values_list[index] is not None and isinstance(values_list[index],str):
                if values_list[index].startswith(prefix):
                    kwargs = KwargsMap.get(prefix, {})
                    data = PrefixMap[prefix](prefix,values_list[index], **kwargs)
                    values_list[index] =data

def _getParamFromFile(prefix,value,mode):
    path=value[len(prefix):]
    path = os.path.expanduser(path)
    path = os.path.os.path.expandvars(path)
    if not os.path.isfile(path):
        errorClass = cliError.error()
        errorMsg='file is not exist'
        errorClass.printInFormat(errorClass.CliException, errorMsg)
    try:
        with open(path, mode) as f:
            data = f.read()
            return data
    except (OSError, IOError) as e:
        print e
def _getParamFromUrl(prefix,value,mode):

    req = urllib2.Request(value)
    try:
        response=urllib2.urlopen(req)
        if response.getcode() == 200:
            return response.read()
        else:
            errorClass = cliError.error()
            errorMsg='Get the wrong content'
            errorClass.printInFormat(response.getcode(), errorMsg)
    except Exception as e:
        print e

PrefixMap = {'file://': _getParamFromFile,
             'fileb://': _getParamFromFile
             #'http://': _getParamFromUrl,
             #'https://': _getParamFromUrl
            }

KwargsMap = {'file://': {'mode': 'r'},
             'fileb://': {'mode': 'rb'},
             #'http://': {},
             #'https://': {}
            }