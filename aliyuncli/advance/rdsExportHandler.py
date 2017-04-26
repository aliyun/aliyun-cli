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

import os,sys,json
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunCliParser
import response
import commandConfigure
import aliyunOpenApiData
import aliyunExtensionCliHandler
import platform

class RdsExportDBInstanceHanlder():
    def __init__(self):
        self.cmd = 'rds'
        self.operations = ['DescribeDBInstanceAttribute']
        self.parser = aliyunCliParser.aliyunCliParser()
        self.apiHandler = aliyunOpenApiData.aliyunOpenApiDataHandler()
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()

    def getFileName(self,keyValues):
        filename = None
        if keyValues.has_key('--filename') and len(keyValues['--filename']) > 0:
            filename = keyValues['--filename'][0]
        else:
            return filename, "A file name is needed! please use \'--filename\' and add the file name."
        return filename, ''

    def exportDBInstance(self, cmd, operation, version, secureRequest=False):
        rdsConfigure = commandConfigure.rds()
        if cmd.lower() == rdsConfigure.cmdName.lower() and operation.lower() ==rdsConfigure.exportDBInstance.lower():
            pass
        else :
            return None
        _keyValues = self.parser.getCliKeyValues()
        filename, errorMsg = self.getFileName(_keyValues)
        if filename is None:
            import cliError
            errorClass = cliError.error()
            errorClass.printInFormat(errorClass.CliException, errorMsg)
            return
        operation = self.operations[0]
        if self.apiHandler.isAvailableOperation(cmd, operation, version):
            cmdInstance, mclassname = self.apiHandler.getInstanceByCmdOperation(cmd, operation, version)
            if not cmdInstance is None:
                newkeyValues = self.parser.getOpenApiKeyValues(_keyValues)
                if self.apiHandler.needSetDefaultRegion(cmdInstance, newkeyValues):
                    newkeyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                self.apiHandler.setAttr(cmdInstance, newkeyValues) # set all key values in instance
                self.apiHandler.changeEndPoint(cmdInstance, newkeyValues)
                try:
                    result = self.apiHandler.getResponse(cmd,operation, mclassname, cmdInstance, newkeyValues,secureRequest)
                    self.apiHandler.responseOptimize(result,cmd,operation)
                    # result = cmdInstance.getResponse()
                    # result = self._optimizeResult(result)
                    if("Code" in result):
                        response.display_response("error", result, "json")
                    else:
                        if not filename == None:
                            self.exportInstanceToFile(result,filename)
                        else:
                            print 'Filename is needed'
                except Exception,e:
                    print(e)

    def exportInstanceToFile(self, result, filename):
        inputFilePath  = os.path.split(filename)[0]
        inputFileName = os.path.split(filename)[1]
        if not os.path.isfile(filename):
            if inputFilePath == '':
                filePath = self.extensionHandler.aliyunConfigurePath
            else:
                filePath = inputFilePath
            fileName=os.path.join(filePath,inputFileName)
            # fileName = os.path.join(filePath, inputFileName)
        else:
            fileName = filename
        fp = open(fileName,'w')
        try :
            fp.write(json.dumps(result,indent=4))
            print "success"
        except IOError:
            print "Error: can\'t find file or read data"
        finally:
            fp.close()

if __name__ == "__main__":
    handler = RdsExportDBInstanceHanlder()
    handler.exportInstanceToFile("haha", "test")

