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

import re
import os
import sys
import aliyun.api
import aliyunExtensionCliHandler
import aliyunCliParser
class aliyunOpenApiDataHandler():
    def __init__(self, path=None):
        self.path = path
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
        self.parser = aliyunCliParser.aliyunCliParser()
        # init app info first
        userKey = ""
        userSecret = ""
        userKey, userSecret = self.parser.getTempKeyAndSecret()
        if userKey is None:
            if not self.extensionHandler.getUserKey() is None:
                userKey = self.extensionHandler.getUserKey()
            else:
                userKey = ""
        if userSecret is None:
            if not self.extensionHandler.getUserSecret() is None:
                userSecret = self.extensionHandler.getUserSecret()
            else:
                userSecret = ""
        # if not self.extensionHandler.getUserKey() is None and userKey is None:
        #     userKey = self.extensionHandler.getUserKey()
        # if not self.extensionHandler.getUserSecret() is None and userSecret is None:
        #     userSecret = self.extensionHandler.getUserSecret()
        aliyun.setDefaultAppInfo(userKey, userSecret)

# this api will return all command from api, such as , ecs, rds, slb
    def getApiCmds(self):
        cmd = set()
        modules = sys.modules.keys()
        for mclass in modules:
            numbers = self.filterNumbers(mclass)
            if numbers.__len__() and mclass.find('aliyun.api') >= 0 and mclass.split('.').__len__() > 3 : # class aliyun.api.rest.classname
                cmd.add(mclass.split(numbers.encode())[0].split('.')[3])
        try:
            for item in ['Bss', 'Yundun']:
                cmd.remove(item)
        except Exception as e:
            pass
        return cmd

    def getApiCmdsLower(self):
        cmds = self.getApiCmds()
        lowerCmds = set()
        for cmd in cmds:
            lowerCmds.add(cmd.lower())
        return lowerCmds

# this api will check if the cmd is the available
    def isAvailableCmd(self, cmdName):
        try:
            apiCmds = self.getApiCmds()
            for cmd in apiCmds:
                if cmdName.lower() == cmd.lower():
                    return True
            return False
        except Exception as e:
            return False

# this api will define all operations from given command
    def getApiOperations(self, command, version):
        operations = set()
        if version.__len__() != 8:
            return operations # 20130110 the date format should equeal 8
        cmds = self.getApiCmds()
        _match = "can not find the operations"
        for cmd in cmds:
            if cmd.lower() == command.lower(): # the cmd comes form cli maybe lower maybe upper
                _match = cmd+version
        modules = sys.modules.keys()
        for mclass in modules:
            if mclass.find('aliyun.api') >= 0 and mclass.split('.').__len__() > 3:
                if mclass.split(_match).__len__() > 1:
                    operations.add(mclass.split(_match)[1].split('Request')[0])
        return operations

# this api will check if the operation is the available
    def isAvailableOperation(self, cmd, operation, version=None):
        if operation is None:
            return False
        if version is None:
            version = self.getLatestVersionByCmdName(cmd)
        operations = self.getApiOperations(cmd,version)
        for item in operations:
            if operation.lower() == item.lower():
                return True
        return False


# this method will give all attr of give class
    def getAttrList(self, classname):
        try:
            # here should be a instance
            return classname.__dict__.keys()
        except Exception as e:
            return None
# this method will set all key:value for open api class
    def setAttr(self, classname, map):
        try:
            for key in map.keys():
                if len(map.get(key)) >= 1 and not key in ["AccessKeyId", "AccessKeySecret", "Endpoint"]:
                    value = map.get(key)[0]
                    key = key.replace("--","")
                    classname.__setattr__(key, value)
        except Exception as e:
            pass

# this method will change the endpoint for api command
    def changeEndPoint(self, classname, keyValues):
        endpoint = "Endpoint"
        try:
            if keyValues.has_key(endpoint) and keyValues[endpoint].__len__() > 0:
                classname._RestApi__domain = keyValues[endpoint][0]
        except Exception as e:
            pass

# this method will check if need set defaut region
    def needSetDefaultRegion(self, classname, map):
        need = True
        try:
            if len(map.get("RegionId")) >= 1: # user has set the regionId
                need = False
        except Exception as e:
            pass
        if need: # user dont give the RegionId and classname has no this attribute
            need = False
            for item in self.getAttrList(classname):
                if item == "RegionId":
                    need = True
        return need


# this method will create a instance by give class name
    def getInstance(self, className):
        if className.find('Request') < 0:
            className = className+"Request"
        moduleName = 'aliyun.api.rest.'+className # here need to change to find the right package
        try:
            module = sys.modules[moduleName]
            mInstance= getattr(module, className)()
            return mInstance
        except Exception as err:
            print err
            return None


# this method create a instance from cli reading cmd + version, should change lower and upper
    def getInstanceByCmd(self, cmdName, operationName, version):
        apiCmds = self.getApiCmds()
        for cmd in apiCmds:
            if cmdName.lower() == cmd.lower():
                cmdName = cmd
        operations = self.getApiOperations(cmdName, version)
        if operationName is None:
            return None
        for item in operations:
            if operationName.lower() == item.lower():
                operationName = item
        className = cmdName+version+operationName
        return self.getInstance(className)

# the following api maybe need to remove
# this api will filter all numbers in one string
    def filterNumbers(self, _string):
        numbers = re.findall(r'(\w[0-9]+)\w*',_string) # this function will return a list
        if isinstance(numbers, list):
            if numbers.__len__() > 0 :
                numbers = ''.join(numbers)
                numbers = list(numbers)
                del numbers[0]
                numbers = ''.join(numbers)
        return numbers

# this api will return all
    def getFiles(self):
        rootPath = os.getcwdu() # here need to change
        files = list()
        apiPath = rootPath+self.path
        for parent, dirnames, filenames in os.walk(apiPath):
            for filename in filenames:
                if filename.endswith('.py'):
                    files.append(filename)
        return files

# this api will return all
    def getFilesByFilter(self, cmdAndVersion):
        rootPath = os.getcwdu() # here need to change
        files = list()
        apiPath = rootPath+self.path
        for parent, dirnames, filenames in os.walk(apiPath):
            for filename in filenames:
                if filename.endswith('.py') and filename.startswith(cmdAndVersion):
                    files.append(filename)
        return files

# this api will get ALL service version
    def getAllServiceVersion(self):
        pass

# this api will get ALL service latest version
    def getAllServiceLatestVersion(self):
        pass

# this api will give the Service latest version by command name
    def getLatestVersionByCmdName(self, cmdName):
        versions = set()
        numberSet = set()
        modules = sys.modules.keys()
        for mclass in modules:
            # here must be .cmdName , avoid any class contain the cmd but not the real class we need.
            if mclass.find('aliyun.api') >= 0 and mclass.lower().find(("."+cmdName).lower()) >= 0 : # class aliyun.api.rest.classname
                numbers = self.filterNumbers(mclass)
                versions.add(int(numbers))
        latestVersion = max(versions)
        return str(latestVersion)

# this api will return the special version from the configure file.
    def getVersionFromCfgFile(self, cmdName):
        pass