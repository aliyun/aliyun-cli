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
import sys,imp,uuid
import aliyunExtensionCliHandler
import aliyunCliParser
import commandConfigure
import aliyunCliConfiugre
from distutils.sysconfig import get_python_lib
import aliyunSdkConfigure
import json
import cliError
import urllib2
import handleEndPoint

from __init__ import  __version__
_userAgent='aliyuncli/'+str(__version__)

version_cmds = ['ConfigVersion','ShowVersions']
END_POINT_OPERATION_LIST = ['ModifyEndPoint']
# import oss
nonStandardSdkCmds = []
try:
    import oss.oss_api
    import oss.oss_util
    import oss.oss_xml_handler
except Exception:
    pass
else:
    nonStandardSdkCmds = ['oss']
    import ossadp.ossHandler





class aliyunOpenApiDataHandler():
    def __init__(self, path=None):
        self.path = path
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
        self.parser = aliyunCliParser.aliyunCliParser()

# this api will return all command from api, such as , ecs, rds, slb
    def getApiCmds(self):
        sitepackages_path=get_python_lib()
        cmds = list()
        cmds.extend(nonStandardSdkCmds)
        sub_objects=os.listdir(sitepackages_path)
        if sub_objects is not None:
            for object in sub_objects:
                if object.startswith('aliyunsdk') and os.path.isdir(os.path.join(sitepackages_path,object)):
                    cmd=object.split('aliyunsdk',1)[1]
                    if len(cmd)>0 and cmd not in['core']:
                        cmds.append(cmd)
        return set(cmds)

    def getApiCmdsLower(self):
        cmds = self.getApiCmds()
        lowerCmds = set()
        for cmd in cmds:
            lowerCmds.add(cmd)
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
#this function is to handle no-POP SDK
    def isNonStandardSdkCmd(self,cmd):
        if cmd in nonStandardSdkCmds:
            return True
        else:
            return False
#this function is to handle no-POP cmd
    def nonStandardSdkCmdHandle(self,cmd):
        if cmd == 'oss':
            self.handleOssCmd(cmd)
#this function is to handle no-POP oss
    def handleOssCmd(self,cmd):
        ossadp.ossHandler.handleOss()

# this api will define all operations from given command
    def getApiOperations(self, command,version):
        operations = []

        if command == 'oss':
            return ossadp.ossHandler.getAvailableOperations()

        sitepackages_path=get_python_lib()
        pre_module='aliyunsdk'
        module=pre_module+command
        sub_path='request'
        request_path=os.path.join(sitepackages_path,module,sub_path)
        version_path=os.path.join(request_path,str(version))
        for root, dirs, files in os.walk(version_path):
            for name in files:
                if name.endswith('Request.py'):
                    operation=name.split('Request.py',1)[0]
                    if len(operation) >0:
                        self.path=root
                        operations.append(operation)
        return set(operations)

    def getInstanceByCmdOperation(self,cmd,operation,version=None):
        if cmd is  None or operation is None:
            return None,None
        else:
            cmdInstance, mclassname = self.getInstance(operation,cmd,version)
            return cmdInstance, mclassname
# this api will return operations which is not comes from aliyun open api.
# parameter : cmd
# cmd will decide which operations need return.

    def getExtensionOperationsFromCmdLower(self, cmd):
        if cmd is None:
            return None
        defaultExtensionOpers=set(['configversion','showversions'])
        if cmd.lower() == "ecs":
            ecsExtensionOpers=set(['exportinstance', 'importinstance'])
            defaultExtensionOpers.update(ecsExtensionOpers)
        if cmd.lower() == "rds":
            rdsConfigure = commandConfigure.rds()
            rdsExtensionOpers = set()
            for item in rdsConfigure.extensionOperations:
                rdsExtensionOpers.add(item.lower())
            defaultExtensionOpers.update(rdsExtensionOpers)
        return defaultExtensionOpers

    def getExtensionOperationsFromCmd(self, cmd):
        if cmd is None and cmd == 'oss':
            return None
        defaultExtensionOpers=set(['ConfigVersion','ShowVersions'])
        if cmd.lower() == "ecs":
            ecsConfigure = commandConfigure.ecs()
            cmdSet = set()
            for item in ecsConfigure.extensionOperations:
                cmdSet.add(item)
            defaultExtensionOpers.update(cmdSet)
        if cmd.lower() == "rds":
            rdsConfigure = commandConfigure.rds()
            cmdSet = set()
            for item in rdsConfigure.extensionOperations:
                cmdSet.add(item)
            defaultExtensionOpers.update(cmdSet)
        return defaultExtensionOpers
			
#check if the operation is ExtensionOperation
    def isAvailableExtensionOperation(self,cmd, operation):
        if operation is None:
            return False
        extension_operations = self.getExtensionOperationsFromCmdLower(cmd)
        if  extension_operations is None:
            return False
        elif operation.lower() in extension_operations:
            return True
        else :
            return False
# this api will check if the operation is the available
    def isAvailableOperation(self, cmd, operation, version=None):
        if operation is None:
            return False
        operations = self.getApiOperations(cmd,version)
        for item in operations:
            #if operation.lower() == item.lower():
            if operation == item:
                return True
        return False


# this method will give all attr of give class
    def getAttrList(self, classname):
        try:
            # here should be a instance
            SetFuncs=[]
            keys=[]
            if classname is not None:
                keys = classname.__dict__.keys()
            for key in keys:
                if key.startswith('set_'):
                    SetFuncs.append(key.replace('set_', ''))
            return SetFuncs
            # return classname.__dict__.keys()
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
        return need


# this method will create a instance by give class name
    def getInstance(self, operation,cmdName,version=None):
        if self.path is None:
            return None
        moduleName=operation+'Request'
        try:
            fp, pathname, desc = imp.find_module(moduleName,[self.path])
            imp.load_module(moduleName, fp, pathname, desc)
            modules_keys=sys.modules.keys()
            for key in modules_keys:
                if key==moduleName:
                    try:
                        module = sys.modules[moduleName]
                        mInstance= getattr(module, moduleName)()
                        className=getattr(module,moduleName)
                        return mInstance,className
                    except Exception as err:
                        print err
        except Exception as err:
            pass
        return None, None

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
    def getAllServiceVersion(self,cmd):
        pass

# this api will get ALL service latest version
    def getAllServiceLatestVersion(self):
        pass


    def hasNecessaryArgs(self,keyValues):
        region_id=self.getRegionId(keyValues)
        userKey=self.extensionHandler.getUserKey()
        userSecret=self.extensionHandler.getUserSecret()
        if region_id is None or userKey is None  or userSecret is None:
            return False
        else:
            return True

    def getResponse(self,cmd,operation,classname,instance,keyValues):
        setFuncs=self.getSetFuncs(classname)
        if len(setFuncs)>0:
            for func in setFuncs:
                key=func.split('set_',1)[1]
                if len(key)>0 and key in keyValues:
                    arg=keyValues[key]
                    if arg is not None and len(arg)>0:
                        param=arg[0]
                        getattr(instance,func)(param)
        userKey=self.getUserKey()
        userSecret=self.getUserSecret()
        regionId=self.getRegionId(keyValues)
        userAgent=self.getUserAgent()
        module='aliyunsdkcore'
        try:
            from aliyunsdkcore import client
            from aliyunsdkcore.acs_exception.exceptions import ClientException as exs
            Client=client.AcsClient(userKey,userSecret,regionId,True,3,userAgent)
            instance.set_accept_format('json')
            result=Client.do_action(instance)
            jsonobj = json.loads(result)
            return jsonobj
        except ImportError as e:
            print module, 'is not exist!'
            return
        except exs as e:
            # print e.get_error_msg()
            error = cliError.error()
            error.printInFormat(e.get_error_code(), e.get_error_msg())
            return


    def getSetFuncs(self,classname):
        SetFuncs=[]
        keys=[]
        if classname is not None:
            keys = classname.__dict__.keys()
        for key in keys:
            if key.startswith('set_'):
                SetFuncs.append(key)
        return SetFuncs

# this api will return the special version from the configure file.
    def getVersionFromCfgFile(self, cmdName):
        pass

    def getAllVersionsByCmdName(self,command):
        versions=[]
        pre_module='aliyunsdk'
        module=pre_module+command
        sitepackages_path=get_python_lib()
        sub_path='request'
        module='aliyunsdk'+command
        request_path=os.path.join(sitepackages_path,module,sub_path)
        objects=os.listdir(request_path)
        for object in objects :
            if object.startswith('v') and os.path.isdir(os.path.join(request_path,object)):
                versions.append(object)
        versions.sort(reverse = True)
        return versions

    def getLatestVersion(self,versions):
        if versions is not None and len(versions)>0:
            return versions[0]

    def getTempVersion(self,keyValues):
        key='--version'
        if keyValues is not None and keyValues.has_key(key):
            return keyValues.get(key)
        key = 'version'
        if keyValues is not None and keyValues.has_key(key):
            return keyValues.get(key)

    def getVersionFromFile(self,cmd):
        version=None
        versionHandler=aliyunSdkConfigure.AliyunSdkConfigure()
        filename=versionHandler.fileName
        version=versionHandler.getCmdVersionFromFile(cmd,filename)
        return version

    def getSdkVersion(self,cmd,keyValues):
        tempVersion=self.getTempVersion(keyValues)
        versions=self.getAllVersionsByCmdName(cmd)
        if tempVersion is not None and len(tempVersion)>0:
            if tempVersion[0] in versions:
                return tempVersion[0]
            else:
                # show error
                error = cliError.error()
                error.printInFormat("Wrong version", "The version input is not exit.")
                return None
        configVersion=self.getVersionFromFile(cmd)
        if configVersion is not None:
            return configVersion
        defaultVersion=self.getLatestVersion(versions)
        if defaultVersion is not None:
            return defaultVersion
    def getUserKey(self):
        userKey=None
        userKey, userSecret = self.parser.getTempKeyAndSecret()
        if userKey is None:
            if self.extensionHandler.getUserKey() is not None:
                userKey = self.extensionHandler.getUserKey()
        return userKey

    def getUserSecret(self):
        userSecret=None
        userKey, userSecret = self.parser.getTempKeyAndSecret()
        if userSecret is None:
            if not self.extensionHandler.getUserSecret() is None:
                userSecret = self.extensionHandler.getUserSecret()
        return userSecret

    def getRegionId(self,keyValues):
        key='RegionId'
        if key in keyValues:
            return keyValues[key][0]
        else:
            return None

    def getUserAgent(self):
        return _userAgent

    def getMacAddress(self):
        node = uuid.getnode()
        mac = uuid.UUID(int = node).hex[-12:]
        return mac

    def responseOptimize(self,response,cmd,operation):
        self.checkForServer(response,cmd,operation)
    def getRequestId(self,response):
        try:
            if response.has_key('RequestId') and len(response['RequestId']) > 0:
                requestId = response['RequestId']
                return  requestId
        except Exception:
            pass

    def checkForServer(self,response,cmd,operation):
        configure = aliyunCliConfiugre.configure()
        requestId = self.getRequestId(response)
        if requestId is None:
            requestId = ""
        ak =  self.getUserKey()
        if ak is None:
            ak = ""
        ua =  self.getUserAgent()
        if ua is None:
            ua = ""
        url = configure.server_url + "?requesId=" + requestId + "&ak=" + ak +"&ua="+ua+"&cmd="+cmd+"&operation="+operation
        try:
            f = urllib2.urlopen(url,data=None,timeout=5)
            s = f.read()
            return s
        except Exception :
            pass

    def isEndPointOperation(self,operation):
        if operation is not None and operation in END_POINT_OPERATION_LIST:
            return True
        else:
            return False

    def handleEndPointOperation(self,cmd,operation,keyValues):
        handleEndPoint.handleEndPoint(cmd,operation,keyValues)

if __name__ == '__main__':
    handler = aliyunOpenApiDataHandler()
    print "###############",handler.isAvailableExtensionOperation('ecs', 'exportInstance')
    print "###############",handler.isAvailableOperation('ecs', 'DescribeInstances')
    print "###############",handler.getExtensionOperationsFromCmd('ecs')
