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
__author__ = 'xixi.xxx'

import aliyunCliHelp
import aliyunOpenApiData

class commandConfigure:
    def __init__(self):
        self.ecs = ecs()
        self.rds = rds()
        self.main_options = ['output', 'AccessKeyId', 'AccessKeySecret', 'Endpoint']
        self.helper = aliyunCliHelp.aliyunCliHelper()
        self.open_api_headler = aliyunOpenApiData.aliyunOpenApiDataHandler()

    def getExtensionOptions(self, cmd, operation):
        if cmd is None or operation is None:
            return None
        if operation in aliyunOpenApiData.version_cmds:
            return None
        if cmd.lower() == 'rds':
            _rds = rds()
            return _rds.extensionOptions[operation]
        if cmd.lower() == 'ecs':
            _ecs = ecs()
            return _ecs.extensionOptions[operation]
        return None

    def getExtensionOperations(self, cmd):
        return self.open_api_headler.getExtensionOperationsFromCmd(cmd)
    #
    # this api will return the options for each extension operations, such as "ecs importInstance" "rds ExportDBInstance"
    #
    def showExtensionOperationHelp(self, cmd, operation):
        parameterList = list()
        if cmd.lower() == "rds":
            self.appendList(parameterList, self.main_options)
            if operation.lower() == self.rds.exportDBInstance.lower():
                # _mlist = self.rds.extensionOptions[self.rds.exportDBInstance]
                self.appendList(parameterList, self.rds.extensionOptions[self.rds.exportDBInstance])
            if operation.lower() == self.rds.importDBInstance.lower():
                # print "haha", (self.rds.extensionOptions[self.rds.importDBInstance])
                # parameterList.append(self.rds.extensionOptions[self.rds.importDBInstance])
                self.appendList(parameterList, self.rds.extensionOptions[self.rds.importDBInstance])

        if cmd.lower() == "ecs":
            self.appendList(parameterList, self.main_options)
            if operation.lower() == self.ecs.exportInstance.lower():
                self.appendList(parameterList, self.ecs.extensionOptions[self.ecs.exportInstance])
            if operation.lower() == self.ecs.importInstance.lower():
                self.appendList(parameterList, self.ecs.extensionOptions[self.ecs.importInstance])

        self.helper.showParameterError(cmd, operation, parameterList)

    def appendList(self, parameterList, optionList):
        for item in optionList:
            parameterList.append(item)

class rds:
    cmdName = 'Rds'
    exportDBInstance = 'ExportDBInstance'
    importDBInstance = 'ImportDBInstance'
    extensionOperations = [exportDBInstance, importDBInstance]
    extensionOptions = {exportDBInstance:['DBInstanceId','OwnerAccount','OwnerId','ResourceOwnerAccount','filename'],
                        importDBInstance:['count','filename']}

class ecs:
    cmdName = 'Ecs'
    exportInstance = 'ExportInstance'
    importInstance = 'ImportInstance'
    extensionOperations = [exportInstance, importInstance]
    extensionOptions = {exportInstance:['InstanceId','OwnerAccount','OwnerId','ResourceOwnerAccount','filename'],
                        importInstance:['count','filename']}

if __name__ == '__main__':
    # print type(rds.extensionOperations)
    # print type(rds.extensionOptions)
    # print rds.extensionOptions['ll']
    configure = commandConfigure()
    print configure.showExtensionOperationHelp("ecs", "ExportInstance")
