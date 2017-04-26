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

import os
import platform
import aliyunCliHelp
import aliyunCliConfiugre
import advance.userProfileHandler
import advance.userConfigHandler
import configure
import advance.ecsExportHandler
import advance.ecsImportHandler
import aliyunCliParser
from __init__ import __version__ as version
class color:
    if platform.system() == "Windows":
        purple = ''
        cyan = ''
        darkcyan = ''
        blue = ''
        green = ''
        yellow = ''
        red  = ''
        bold = ''
        underline = ''
        end = ''
    else:
        purple = '\033[95m'
        cyan = '\033[96m'
        darkcyan = '\033[36m'
        blue = '\033[94m'
        green = '\033[92m'
        yellow = '\033[93m'
        red  = '\033[91m'
        bold = '\033[1m'
        underline = '\033[4m'
        end = '\033[0m'

class AliyunCredentials:
    aliyun_access_key_id = "aliyun_access_key_id"
    aliyun_access_key_secret = "aliyun_access_key_secret"

class aliyunExtensionCliHandler:
    def __init__(self):
        self._version = version
        self.home = ".aliyuncli"
        self.configure = "configure"
        self.credentials = "credentials"
        self.aliyunConfigurePath = os.path.join(self.findConfigureFilePath(),self.home)
        self.globalConfigure = aliyunCliConfiugre.configure()
        self.parser = aliyunCliParser.aliyunCliParser()
    def getExtensionCmd(self):
        return None

# this function will give all extension command defined by us
    def getAllExtensionCommands(self):
        cmds = list()
        cmds = ['-h', '--help', 'help', 'configure', '--version', 'version', 'useprofile', 'addprofile','showconfig']
        return cmds
# this function will handler extension operation
    def handlerExtensionOperation(self,cmd,operation,version,secureRequest = False):
        defaultOperations=['configversion','showversions']
        if operation.lower() in defaultOperations:
            import aliyunSdkConfigure
            aliyunSdkConfigure.handleSdkVersion(cmd,operation,version)
        if cmd.lower() == 'ecs':
            import commandConfigure
            import advance.ecsExportHandler
            import advance.ecsImportHandler
            ecsConfigure = commandConfigure.ecs()
            _ecsImportHandler = advance.ecsImportHandler.EcsImportHandler()
            _ecsExportHandler = advance.ecsExportHandler.EcsExportHandler()
            if  operation.lower() == ecsConfigure.importInstance.lower():
                _ecsImportHandler.ImportInstance(cmd,operation,version,secureRequest)
            elif operation.lower() == ecsConfigure.exportInstance.lower():
                _ecsExportHandler.exportInstance(cmd,operation,version,secureRequest)

        if cmd.lower() == 'rds':
            import commandConfigure
            import advance.rdsExportHandler
            import advance.rdsImportHandler
            rdsConfigure = commandConfigure.rds()
            rdsExportDBInstanceHandler = advance.rdsExportHandler.RdsExportDBInstanceHanlder()
            rdsImportDBInstanceHanlder = advance.rdsImportHandler.RdsImportDBInstanceHandler()
            if operation.lower() == rdsConfigure.exportDBInstance.lower():
                rdsExportDBInstanceHandler.exportDBInstance(cmd,operation,version,secureRequest)
            elif operation.lower() == rdsConfigure.importDBInstance.lower():
                rdsImportDBInstanceHanlder.importInstance(cmd, operation, version,secureRequest)
		
# this function will handler extension command
    def handlerExtensionCmd(self, cmd):
        _cmd = self.parser._getCommand()
        _keyvalues = self.parser._getKeyValues()
        operation = self.parser._getOperations()
        if cmd.lower() in ['help', '-h', '--help']:
            self.showAliyunCliHelp()
        if cmd.lower() == 'configure':
            configure.handleConfigure(cmd,operation)
        if cmd.lower() == 'useprofile':
            _profileHandler = advance.userProfileHandler.ProfileHandler()
            _profileHandler.handleProfileCmd(_cmd, _keyvalues)
        if cmd.lower() == 'addprofile':
            _profileHandler = advance.userProfileHandler.ProfileHandler()
            _profileHandler.addProfileCmd(_cmd, _keyvalues)
        if cmd.lower() in ['--version', 'version']:
            self.showCurrentVersion()
        if cmd.lower() == 'showconfig':
            _configHandler = advance.userConfigHandler.ConfigHandler()
            _configHandler.showConfig()
		 

# this api will show help page when user input aliyuncli help(-h or --help)
    def showAliyunCliHelp(self):
        print color.bold+"ALIYUNCLI()"+color.end
        print color.bold+"\nNAME"+color.end
        print "\taliyuncli -"
        print color.bold+"\nDESCRIPTION"+color.end
        print "\tThe Aliyun Command Line Interface is a unified tool to manage your aliyun services. "
        print color.bold+"\nSYNOPSIS"+color.end
        print "\taliyuncli <command> <operation> [options and parameters]"
        print "\n\taliyuncli has supported command completion now. The detail you can check our site."
        print color.bold+"OPTIONS"+color.end
        print color.bold+"\tconfigure"+color.end
        print "\n\tThis option will help you save the key and secret and your favorite output format (text, json or table)"
        print color.bold+"\n\t--output"+color.end+" (string)"
        print "\n\tThe formatting style for command output."
        print "\n\to json"
        print "\n\to text"
        print "\n\to table"
        
        print color.bold+"\n\t--secure"+color.end
        print "\n\tMaking secure requests(HTTPS) to service"
        
        print color.bold+"\nAVAILABLE SERVICES"+color.end
        print "\n\to ecs"
        print "\n\to ess"
        print "\n\to mts"
        print "\n\to rds"
        print "\n\to slb"

    def showCurrentVersion(self):
        print self._version

    def findConfigureFilePath(self):
        homePath = ""
        if platform.system() == "Windows":
            homePath = os.environ['HOMEPATH']
            pass
        else:
            homePath = os.environ['HOME']
            pass
        return homePath
    def getUserRegion(self):
        handler = configure.AliyunConfig()
        return handler._getUserRegion()

    def getUserFormat(self):
        handler = configure.AliyunConfig()
        return handler._getUserFormat()
    def getUserKey(self):
        handler = configure.AliyunConfig()
        return handler._getUserKey()

    def getUserSecret(self):
        handler = configure.AliyunConfig()
        return  handler._getUserSecret()

    def getPort(self):
        handler = configure.AliyunConfig()
        return  handler._getPort()
if __name__ == "__main__":
    pass
# this function will show aliyun cli help
