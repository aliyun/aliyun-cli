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

import aliyunOpenApiData
import cliError
class aliyunCliHelper:
    def __init__(self):
        self.openApiDataHandler = aliyunOpenApiData.aliyunOpenApiDataHandler()

    def showUsage(self):
        print "usage: aliyuncli <command> <operation> [options and parameters]"

    def showExample(self):
        print "show example"

    def showCmdError(self, cmd):
        self.showUsage()
        print "<aliyuncli> the valid command as follows:\n"
        cmds = self.openApiDataHandler.getApiCmds()
        self.printAsFormat(cmds)

    def showOperationError(self, cmd, operation):
        keyValues=self.openApiDataHandler.parser._getKeyValues()
        version = self.openApiDataHandler.getSdkVersion(cmd,keyValues)
        versions=self.openApiDataHandler.getAllVersionsByCmdName(cmd)
        if version not in versions:
                error = cliError.error()
                error.printInFormat("Wrong version", "The sdk version is not exit.")
                return None
        self.showUsage()
        print "["+cmd+"]","valid operations as follows:\n"
        operations = self.openApiDataHandler.getApiOperations(cmd, version)
        extensions = self.openApiDataHandler.getExtensionOperationsFromCmd(cmd)
        operations.update(extensions)
        import commandConfigure
        if cmd.lower() == 'rds':
            rdsConfigure = commandConfigure.rds()
            operations.add(rdsConfigure.exportDBInstance)
            operations.add(rdsConfigure.importDBInstance)
        self.printAsFormat(operations)

    def showParameterError(self, cmd, operation, parameterlist):
        print 'usage: aliyuncli <command> <operation> [options and parameters]'
        print '['+cmd+"."+operation+']: current operation can uses parameters as follow :\n'
        self.printAsFormat(parameterlist)
        pass

    def printAsFormat(self, data):
        mlist = list()
        for item in data:
            mlist.append(item)
        mlist.sort()
        count = 0
        tmpList = list()
        for item in mlist:
            tmpList.append(item)
            count = count+1
            if len(tmpList) == 2:
                print '{0:40}'.format(tmpList[0]),'\t|',format(tmpList[1],'<10')
                tmpList = list()
            if len(tmpList) == 1 and count == len(mlist):
                print tmpList[0]