import os,sys,json
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunExtensionCliHandler
import aliyunOpenApiData
import aliyunCliParser
import response 
class EcsExportHandler:
    def __init__(self):
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
    
    def exportInstance(self,cmd,operation,version,secureRequest = False):
        if cmd.lower() == 'ecs' and operation.lower() =='exportinstance':
            operations = ['DescribeInstanceAttribute']
        else :
            return None
        _keyValues = self.parser.getCliKeyValues()
        # filename = self.getFileName(_keyValues)
        filename, errorMsg = self.getFileName(_keyValues)
        if filename is None:
            import cliError
            errorClass = cliError.error()
            errorClass.printInFormat(errorClass.CliException, errorMsg)
            return
        for item in operations:
            if self.apiHandler.isAvailableOperation(cmd, item, version):
                cmdInstance, mclassname = self.apiHandler.getInstanceByCmdOperation(cmd, item, version)
                if not cmdInstance is None:
                    newkeyValues = self.parser.getOpenApiKeyValues(_keyValues)
                    if self.apiHandler.needSetDefaultRegion(cmdInstance, newkeyValues):
                        newkeyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                    self.apiHandler.setAttr(cmdInstance, newkeyValues) # set all key values in instance
                    self.apiHandler.changeEndPoint(cmdInstance, newkeyValues)
                    try:
                        #result = cmdInstance.getResponse()
                        result = self.apiHandler.getResponse(cmd,operation, mclassname, cmdInstance, newkeyValues,secureRequest)
                        result = self._optimizeResult(result)
                        self.apiHandler.responseOptimize(result,cmd,operation)
                        if("Code" in result):
                            response.display_response("error", result, "json")
                        else:
                            if not filename == None:
                                self.exportInstanceToFile(result,filename)
                            else:
                                print 'Filename is needed'
                    except Exception,e:
                        print(e)
    def _optimizeResult(self,result):
        keys = result.keys()
        if 'SecurityGroupIds' in keys and 'SecurityGroupId' in result['SecurityGroupIds'] :
            SGIds = result['SecurityGroupIds']['SecurityGroupId'][0]    
            result['SecurityGroupId'] = SGIds
        rubbishKeys = ('SecurityGroupIds','CreationTime','PublicIpAddress','VpcAttributes','Status',\
'EipAddress','SerialNumber','OperationLocks','RequestId','InnerIpAddress',)
        for item in keys:
            if item in rubbishKeys:
                del result[item]
        return result
    def exportInstanceToFile(self,result,filename):
        inputFilePath  = os.path.split(filename)[0]
        inputFileName = os.path.split(filename)[1]
        if not os.path.isfile(filename):
            if inputFilePath == '':
                filePath = self.extensionHandler.aliyunConfigurePath
            else:
                filePath = inputFilePath
            # fileName = filePath + inputFileName
            fileName = os.path.join(filePath, inputFileName)
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
    handler = EcsExportHandler()
    handler.exportInstanceToFile()

