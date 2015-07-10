__author__ = 'zhaoyang.szy'
import os,sys
import response 
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunExtensionCliHandler
class ConfigCmd:
    showConfig = 'showConfig'
    importConfig = 'importConfig'
    exportConfig = 'exportConfig'
    name = '--filename'
class ConfigHandler:
    def __init__(self):
        self.extensionCliHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
    def getConfigHandlerCmd(self):
        return [ConfigCmd.showConfig,ConfigCmd.importConfig,ConfigCmd.exportConfig]

    def getConfigHandlerOptions(self):
        return [ConfigCmd.name]
				
    def showConfig(self):
        _credentialsPath = os.path.join(self.extensionCliHandler.aliyunConfigurePath,self.extensionCliHandler.credentials)
        _configurePath = os.path.join(self.extensionCliHandler.aliyunConfigurePath,self.extensionCliHandler.configure)
        config = dict()
        configContent = dict() 
	credentialsContent = dict ()
	if os.path.exists(_configurePath):
            for line in open(_configurePath):
                line = line.strip('\n')
                if line.find('=') > 0:
                    list = line.split("=",1)
		    configContent[list[0]] = list[1]
		else: 
		    pass
	config['configure'] = configContent
	if os.path.exists(_credentialsPath):
	    for line in open(_credentialsPath):
                line = line.strip('\n')
                if line.find('=') > 0:
                    list = line.split("=",1)
		    credentialsContent[list[0]] = list[1]
		else: 
		    pass 
	config ['credentials'] = credentialsContent
	response.display_response("showConfigure",config,'table')
    def importConfig():
        pass
    def exportConfig():
        pass
	


if __name__ == "__main__":
    handler = ConfigHandler()
    handler.showConfig()
