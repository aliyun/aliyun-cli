import os,sys,json
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunExtensionCliHandler
import aliyunOpenApiData
import aliyunCliParser
import response 
class RdsImportDBInstanceHandler:
    def __init__(self):
        self.parser = aliyunCliParser.aliyunCliParser()
        self.apiHandler = aliyunOpenApiData.aliyunOpenApiDataHandler()
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
    
    def getFileName(self,keyValues):
        filename = None
        if keyValues.has_key('--filename') and len(keyValues['--filename']) > 0:
            filename = keyValues['--filename'][0]
        else:
            return filename, "A filename is needed! please use \'--filename\' and add the file name."
        return filename, ""
    def getInstanceCount(self,keyValues):
        count = 1
        import_count = "--count"
        if keyValues.has_key(import_count) and len(keyValues[import_count]) > 0:
            if  keyValues[import_count][0].isdigit() and int(keyValues[import_count][0]) >= 0:
                count = keyValues[import_count][0]
            else:
                pass
                # print "InstanceCount should be a positive number! The default value(1) will be used!"
        return int(count), "InstanceCount is "+str(count)+" created."
    
    def getSubOperations(self,cmd,operation):
        import commandConfigure
        _rds = commandConfigure.rds()
        operations = None
        if cmd.lower() == 'rds' and operation.lower() ==_rds.importDBInstance.lower():
            operations = ['CreateDBInstance']
        return operations 

    def _handSubOperation(self,cmd,operations,keyValues,version,secureRequest=False):
        for item in operations:
            if self.apiHandler.isAvailableOperation(cmd, item, version):
                cmdInstance, mclassname = self.apiHandler.getInstanceByCmdOperation(cmd, item, version)
                if not cmdInstance is None:
                    newkeyValues = self.parser.getOpenApiKeyValues(keyValues)
                    if self.apiHandler.needSetDefaultRegion(cmdInstance, newkeyValues):
                        newkeyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                    newkeyValues["ClientToken"] = [self.random_str()]
                    # print newkeyValues.keys()
                    # return
                    # self._setAttr(cmdInstance, newkeyValues) # set all key values in instance
                    # self.apiHandler.changeEndPoint(cmdInstance, newkeyValues)
                    try:
                        result = self.apiHandler.getResponse(cmd, item, mclassname, cmdInstance, newkeyValues,secureRequest)
                        self.apiHandler.responseOptimize(result,cmd,item)
                        # result = cmdInstance.getResponse()
                        if("Code" in result):
                            response.display_response("error", result, "json")
                        else:
                            response.display_response(item, result, "json")
                    except Exception,e:
                        print(e)
   
    def getKVFromJson(self,filename):
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
        try:
            fp = open(fileName,'r')
            data=json.loads(fp.read())
            keys = data.keys()
            # print keys, type(data['Items']['DBInstanceAttribute'][0])
            # instanceAttribute = data['Items']['DBInstanceAttribute'][0]
            items = data['Items']['DBInstanceAttribute'][0]
            keys = items.keys()
            newMap = dict()
            for key in keys:
                value = list()
                value.append(items[key])
                newMap[key] = value
            return newMap
            # return data['Items']['DBInstanceAttribute'][0]
        finally:
            fp.close()

    def importInstance(self,cmd,operation,version,secureRequest = False):
        _keyValues = self.parser.getCliKeyValues()
        operations = self.getSubOperations(cmd,operation)
        _instanceCount, countMsg = self.getInstanceCount(_keyValues)
        filename, errorMsg = self.getFileName(_keyValues)
        if filename is None:
            import cliError
            errorClass = cliError.error()
            errorClass.printInFormat(errorClass.CliException, errorMsg)
            return
        keyValues = self.getKVFromJson(filename)
        keyValues['PayType'] = ["Postpaid"]
        for i in range(1,_instanceCount+1):
            self._handSubOperation(cmd,operations,keyValues,version,secureRequest)

    # this method will set all key:value for open api class
    def _setAttr(self, classname, map):
        try:
            for key in map.keys():
                if  not key in ["AccessKeyId", "AccessKeySecret", "Endpoint"]:
                    value = map.get(key)
                    classname.__setattr__(key, value)
        except Exception as e:
            pass

    def random_str(self, randomlength=30):
        from random import Random
        str = ''
        chars = 'AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz0123456789'
        length = len(chars) - 1
        random = Random()
        for i in range(randomlength):
            str += chars[random.randint(0, length)]
        return str
 

if __name__ == "__main__":
    handler = RdsImportDBInstanceHandler()
    # handler.getKVFromJson('ttt')
    # print handler.getKVFromJson('ttt')
    print handler.random_str()


