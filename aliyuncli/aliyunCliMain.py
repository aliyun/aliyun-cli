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

import aliyunCliParser
import aliyunOpenApiData
import aliyunExtensionCliHandler
import response
import aliyunCliHelp
import aliyunCliUpgrade
import aliyunCompleter
import sys

class AliyunCommandLine:
    def __init__(self):
        self.parser = aliyunCliParser.aliyunCliParser()
        self.handler = aliyunOpenApiData.aliyunOpenApiDataHandler()
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
        self.helper = aliyunCliHelp.aliyunCliHelper()
        self.args = sys.argv[1:]
        self.completer = aliyunCompleter.Completer()

    def main(self):

        # fetch command
        cmd = self.parser.getCliCmd()
        extensionCmdList = self.extensionHandler.getAllExtensionCommands()

        if cmd in extensionCmdList:
            self.handlerExtensionCmd(cmd)
            return

        # fetch operation
        operation = self.parser.getCliOperation()

        # fetch paramlist
        keyValues = self.parser._getKeyValues()
        
        outPutFormat = self.parser.getOutPutFormat(keyValues)
        if outPutFormat is None or len(outPutFormat) == 0:
            outPutFormat = self.extensionHandler.getUserFormat()
            if outPutFormat is None or outPutFormat == "":
                outPutFormat = 'json'
        else:
            outPutFormat = outPutFormat[0]

        secureRequest = self.parser.getSecureChoice(keyValues)

        if self.handler.isEndPointOperation(operation):
            keyValues = self.parser.getOpenApiKeyValues(keyValues)
            self.handler.handleEndPointOperation(cmd,operation,keyValues)
            return

        if self.handler.isAvailableCmd(cmd):
            # fetch getversion
            if self.handler.isNonStandardSdkCmd(cmd):
                self.handler.nonStandardSdkCmdHandle(cmd)
                return
            version=self.handler.getSdkVersion(cmd,keyValues)
            if version is None:
                return
            # here handler the openapi cmd
            if self.handler.isAvailableOperation(cmd, operation,version): # cmd and operation both are right
                instanceAndClassName=self.handler.getInstanceByCmdOperation(cmd, operation,version)
                if instanceAndClassName is not None and len(instanceAndClassName)==2:
                    cmdInstance = instanceAndClassName[0]
                    className = instanceAndClassName[1]
                    if cmdInstance is not None and className is not None:
                        if self.showInstanceAttribute(cmd, operation, className):
                            return
                        # here should handle the keyValues first
                        keyValues = self.parser.getOpenApiKeyValues(keyValues)
                        if self.handler.needSetDefaultRegion(cmdInstance, keyValues):
                            keyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                        #check necessaryArgs as:accesskeyid accesskeysecret regionId
                        if not self.handler.hasNecessaryArgs(keyValues):
                            print 'accesskeyid/accesskeysecret/regionId is absence'
                            return
                        try:
                            result = self.handler.getResponse(cmd,operation,className,cmdInstance,keyValues,secureRequest)
                            if result is None:
                                return
                            self.handler.responseOptimize(result,cmd,operation)
                            if("Code" in result):
                                response.display_response("error", result, "json")
                                # print("failed")
                                # print(result["Code"])
                                # print(result["Message"])
                                # print("Please check your parameters first.")
                            else:
                                #print(result)
                                response.display_response(operation, result, outPutFormat,keyValues)
                        except Exception as e:
                            print(e)
                    else:
                        print 'aliyuncli internal error, please contact: zikuan.ly@alibaba-inc.com'
            elif self.handler.isAvailableExtensionOperation(cmd, operation):
                if self.args.__len__() >= 3 and self.args[2] == 'help':
                    import commandConfigure
                    configure = commandConfigure.commandConfigure()
                    configure.showExtensionOperationHelp(cmd, operation)
                else:
                    self.extensionHandler.handlerExtensionOperation(cmd,operation,version,secureRequest)
                # self.extensionHandler.handlerExtensionOperation(cmd,operation,version)
            else:
                # cmd is right but operation is not right
                self.helper.showOperationError(cmd, operation)
        else:
            self.helper.showCmdError(cmd)



    def handlerExtensionCmd(self, cmd):
        self.extensionHandler.handlerExtensionCmd(cmd)

    def showInstanceAttribute(self, cmd, operation, classname):
        if self.args.__len__() >= 3 and self.args[2] == "help":
            self.helper.showParameterError(cmd, operation, self.completer._help_to_show_instance_attribute(classname))
            #print self.completer._help_to_show_instance_attribute(cmdInstance)
            return True
        return False








