import os,sys,json
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunExtensionCliHandler
import aliyunOpenApiData
import aliyunCliParser
import response 
class EcsImportHandler:
    def __init__(self):
        self.parser = aliyunCliParser.aliyunCliParser()
        self.apiHandler = aliyunOpenApiData.aliyunOpenApiDataHandler()
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
    
    def getFileName(self,keyValues):
        filename = None
        if keyValues.has_key('--filename') and len(keyValues['--filename']) > 0:
            filename = keyValues['--filename'][0]
        else:
            print "A profile is needed! please use \'--filename\' and add the profile name."
        return filename
    def getInstanceCount(self,keyValues):
        count = 1
        if keyValues.has_key('--instancecount') and len(keyValues['--instancecount']) > 0:
            if  keyValues['--instancecount'][0].isdigit() and int(keyValues['--instancecount'][0]) >= 0:
                count = keyValues['--instancecount'][0]
            else:
                print "InstanceCount should be a positive number! The default value(1) will be used!"
        return int(count)
    
    def getSubOperations(self,cmd,operation):
        operations = None
        if cmd.lower() == 'ecs' and operation.lower() =='importinstance':
            operations = ['CreateInstance']
        return operations 

    def _handSubOperation(self,cmd,operations,keyValues,version):
        for item in operations:
            if self.apiHandler.isAvailableOperation(cmd, item, version):
                cmdInstance,mclassname = self.apiHandler.getInstanceByCmdOperation(cmd, item, version)
                if not cmdInstance is None:
                    newkeyValues = self.parser.getOpenApiKeyValues(keyValues)
                    if self.apiHandler.needSetDefaultRegion(cmdInstance, newkeyValues):
                        newkeyValues["RegionId"] = [self.extensionHandler.getUserRegion()]
                    # self._setAttr(cmdInstance, newkeyValues) # set all key values in instance
                    # self.apiHandler.changeEndPoint(cmdInstance, newkeyValues)
                    try:
                        # result = cmdInstance.getResponse()
                        result = self.apiHandler.getResponse(cmd, item, mclassname, cmdInstance, newkeyValues)
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
            fileName = os.path.join(filePath, filename)
        else:
            fileName = filename
        fp = open(fileName,'r')
        try:
            data=json.loads(fp.read())
            keys = data.keys()
            newMap = dict()
            for key in keys:
                value = list()
                value.append(data[key])
                newMap[key] = value
            return newMap	
        finally:
            fp.close()

    def ImportInstance(self,cmd,operation,version):
        _keyValues = self.parser.getCliKeyValues()
        operations = self.getSubOperations(cmd,operation)
        _instanceCount = self.getInstanceCount(_keyValues)
        filename = self.getFileName(_keyValues)
        keyValues = self.getKVFromJson(filename)
        for i in range(1,_instanceCount+1):
            self._handSubOperation(cmd,operations,keyValues,version)
# this method will set all key:value for open api class
    def _setAttr(self, classname, map):
        try:
            for key in map.keys():
                if  not key in ["AccessKeyId", "AccessKeySecret", "Endpoint"]:
                    value = map.get(key)
                    classname.__setattr__(key, value)
        except Exception as e:
            pass
 

if __name__ == "__main__":
    handler = EcsImportHandler()
    handler.getKVFromJson('ttt')
    print handler.getKVFromJson('ttt')

