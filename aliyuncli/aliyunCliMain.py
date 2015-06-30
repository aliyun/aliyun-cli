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
        self.upgradeHandler = aliyunCliUpgrade.aliyunCliUpgradeHandler()
        self.args = sys.argv[1:]
        self.completer = aliyunCompleter.Completer()

    def main(self):

        # no matter what error happened , should not interrupt user execute the cmd
        # we must keep this code pass and cmdline can running normally.
        try:
            isNewVersion, url = self.upgradeHandler.checkForUpgrade()
            if isNewVersion:
                print "\nAliyuncli new version is ready, you can download the package from the www.aliyun.com.\n"
                choice = raw_input("Notify later ? [Y: notify later; N:I have known it]:")
                # print choice
                self.upgradeHandler.handleUserChoice(choice)
        except Exception as e:
            pass
        finally:
            pass

        # fetch command
        cmd = self.parser.getCliCmd()
        extensionCmdList = self.extensionHandler.getAllExtensionCommands()

        if cmd in extensionCmdList:
            self.handlerExtensionCmd(cmd)
            return

        # fetch operation
        operation = self.parser.getCliOperation()

        # fetch paramlist
        keyValues = self.parser.getCliKeyValues()

        outPutFormat = self.parser.getOutPutFormat(keyValues)
        if outPutFormat is None or len(outPutFormat) == 0:
            outPutFormat = self.extensionHandler.getUserFormat()
            if outPutFormat is None or outPutFormat == "":
                outPutFormat = 'json'
        else:
            outPutFormat = outPutFormat[0]

        if self.handler.isAvailableCmd(cmd):
            # here handler the openapi cmd
            if self.handler.isAvailableOperation(cmd, operation): # cmd and operation both are right
                # fetch getversion
                version = self.handler.getLatestVersionByCmdName(cmd)
                cmdInstance = self.handler.getInstanceByCmd(cmd, operation, version)
                if not cmdInstance is None:
                    if self.showInstanceAttribute(cmd, operation, cmdInstance):
                        return
                    # here should handle the keyValues first
                    keyValues = self.parser.getOpenApiKeyValues(keyValues)
                    if self.handler.needSetDefaultRegion(cmdInstance, keyValues):
                        keyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                    self.handler.setAttr(cmdInstance, keyValues) # set all key values in instance
                    #print "domain:", cmdInstance._RestApi__domain , keyValues
                    self.handler.changeEndPoint(cmdInstance, keyValues)
                    #print "domain:", cmdInstance._RestApi__domain
                    try:
                        result = cmdInstance.getResponse()
                        if("Code" in result):
                            response.display_response("error", result, "json")
                            # print("failed")
                            # print(result["Code"])
                            # print(result["Message"])
                            # print("Please check your parameters first.")
                        else:
                            #print(result)
                            response.display_response(operation, result, outPutFormat)
                    except Exception,e:
                        print(e)
                else:
                    print 'aliyuncli internal error, please contact: xixi.xxx'
            else:
                # cmd is right but operation is not right
                self.helper.showOperationError(cmd, operation)
        else:
            self.helper.showCmdError(cmd)



    def handlerExtensionCmd(self, cmd):
        self.extensionHandler.handlerExtensionCmd(cmd)

    def showInstanceAttribute(self, cmd, operation, cmdInstance):
        if self.args.__len__() >= 3 and self.args[2] == "help":
            self.helper.showParameterError(cmd, operation, self.completer._help_to_show_instance_attribute(cmdInstance))
            #print self.completer._help_to_show_instance_attribute(cmdInstance)
            return True
        return False








