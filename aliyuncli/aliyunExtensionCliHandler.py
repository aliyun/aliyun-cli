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
        self.tag = "/"
        if platform.system() == "Windows":
            self.tag = "\\"
        self.aliyunConfigurePath = self.findConfigureFilePath()+self.tag+self.home+self.tag
        self.globalConfigure = aliyunCliConfiugre.configure()
    def getExtensionCmd(self):
        return None

# this function will give all extension command defined by us
    def getAllExtensionCommands(self):
        cmds = list()
        cmds = ['-h', '--help', 'help', 'configure', '--version', 'version']
        return cmds

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
        print color.bold+"\nAVAILABLE SERVICES"+color.end
        print "\n\to ecs"
        print "\n\to ess"
        print "\n\to mts"
        print "\n\to rds"
        print "\n\to slb"




# this function will handler extension command
    def handlerExtensionCmd(self, cmd):
        if cmd.lower() in ['help', '-h', '--help']:
            self.showAliyunCliHelp()
        if cmd.lower() == 'configure':
            self.handleConfigureCmd()
            # print self.getUserFormat()
            # print self.getUserKey()
            # print self.getUserSecret()
        if cmd.lower() in ['--version', 'version']:
            self.showCurrentVersion()

    def showCurrentVersion(self):
        print self._version

    name2prompt = [
        ('aliyun_access_key_id', "Aliyun Access Key ID"),
        ('aliyun_access_key_secret', "Aliyun Access Key Secret"),
        ('region', "Default Region Id"),
        ('output', "Default output format"),
    ]

    def handleConfigureCmd(self):
        if not os.path.exists(self.aliyunConfigurePath):
            os.makedirs(self.aliyunConfigurePath)
        config = self.getConfig()
        
        if not os.path.isfile(self.aliyunConfigurePath+self.configure):
            _configFile = open(self.aliyunConfigurePath+self.configure, 'w')
            try:
                configtxt = "[default]\n"
                configtxt = configtxt + "ignore=no\n"
                _configFile.write(configtxt)
            finally:
                _configFile.close()
        if not os.path.isfile(self.aliyunConfigurePath+self.credentials):
            _credentialsFile = open(self.aliyunConfigurePath+self.credentials, 'w')
            try:
                credentialstxt = "[default]\n"
                _credentialsFile.write(credentialstxt)
            finally:
                _credentialsFile.close()
        new_values = {}
        if config['ignore'] is None:
            new_values['ignore'] = 'no'
        for name, prompt in self.name2prompt:
            value = config.get(name)
            new_value = self.get_value(value, name,prompt)
            if new_value is not None and new_value != value:
                new_values[name] = new_value
        config_filename = self.getConfigFileName()
        creds_filename = self.getCredsFileName()
        if new_values:
            self._writeCredsToFile(new_values,creds_filename)
            self._updateConfig(new_values,config_filename)


    def getConfig(self):
        config = {}
        ignoreValue = self.getIgnoreValues()
        if ignoreValue == '':
            ignoreValue = None
        userRegion = self.getUserRegion()
        if userRegion == '':
            userRegion = None
        userFormat = self.getUserFormat()
        userKey = self.getUserKey()
        userSecret = self.getUserSecret()
        config['ignore'] = ignoreValue
        config['output'] = userFormat
        config['region'] = userRegion
        config['aliyun_access_key_secret'] = userSecret
        config['aliyun_access_key_id'] = userKey
        return config


    def getConfigFileName(self):
        configFileName = self.aliyunConfigurePath+self.configure
        return configFileName

    def getCredsFileName(self):
        credsFileName = self.aliyunConfigurePath+self.credentials
        return credsFileName

    def _writeCredsToFile(self,new_values,creds_filename):
        credential_file_values = {}
        if 'aliyun_access_key_id' in new_values:
            credential_file_values['aliyun_access_key_id'] = new_values.pop(
                'aliyun_access_key_id')
        if 'aliyun_access_key_secret' in new_values:
            credential_file_values['aliyun_access_key_secret'] = new_values.pop(
                'aliyun_access_key_secret')
        if credential_file_values:
            if creds_filename is not None:
                self._updateConfig(credential_file_values,creds_filename)

    def _updateConfig(self,new_values,config_filename):
        contents = []
        try:
            with open(config_filename, 'r') as f:
                contents = f.readlines()
            self._updateContents(contents, new_values)
            with open(config_filename, 'w') as f:
                f.write(''.join(contents))
        finally:
            f.close()

    def _updateContents(self,contents,new_values):
        new_values = new_values.copy()
        j = 0
        while j < len(contents):
            line = contents[j]
            key_name = self.getKey(line)
            if key_name in new_values:
                value = new_values[key_name]
                new_line = '%s=%s\n' %(key_name,value)
                contents[j] = new_line
                del new_values[key_name]
            j = j+1
        if new_values:
            self._insertNewValues(contents,new_values)
    
    def getKey(self,line):
        key = None
        if line is not None and  line.find('=') >0 :
            key = line.split("=",1)[0]
        return key
    
    def _insertNewValues(self,contents,new_values):
        for key, value in list(new_values.items()):
            contents.append('%s=%s\n' % (key,value))
            del new_values[key]


    def get_value(self, current_value, config_name, prompt_text=''):
        if config_name in ('aliyun_access_key_id', 'aliyun_access_key_secret'):
            if current_value != '' and not current_value is None:
                current_value = self._mask_value(current_value)
        response = raw_input("%s [%s]: " % (prompt_text, current_value))
        if response is '':
            response = None
        return response

    def _mask_value(self,current_value):
        if current_value is None:
            return 'None'
        else:
            return ('*' * 16) + current_value[-4:]

    def findConfigureFilePath(self):
        homePath = ""
        if platform.system() == "Windows":
            homePath = os.environ['HOMEPATH']
            pass
        else:
            homePath = os.environ['HOME']
            pass
        return homePath

    def getIgnoreValues(self):
        configurePath = self.aliyunConfigurePath+self.configure
        if os.path.exists(configurePath):
            f = open(configurePath, 'r')
            _ignore = ""
            try:
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.find('ignore') >=0 :
                        if len(line.split("=",1)) == 2:
                            _ignore = line.split("=",1)[1].replace("\n","")
                return _ignore
            finally:
                f.close()

    def getUserRegion(self):
        configurePath = self.aliyunConfigurePath+self.configure
        if os.path.exists(configurePath):
            f = open(configurePath, 'r')
            _region = ""
            try:
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.find('region') >=0 :
                        if len(line.split("=",1)) == 2:
                            _region = line.split("=",1)[1].replace("\n","")
                return _region
            finally:
                f.close()

    def getUserFormat(self):
        configurePath = self.aliyunConfigurePath+self.configure
        if os.path.exists(configurePath):
            f = open(configurePath, 'r')
            _format = None 
            try:
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.find('output') >=0 :
                        if len(line.split("=",1)) == 2:
                            _format = line.split("=",1)[1].replace("\n","")
                return _format
            finally:
                f.close()

    def getUserKey(self):
        credentials = self.aliyunConfigurePath+self.credentials
        if os.path.exists(credentials):
            f = open(credentials, 'r')
            key = None
            try:
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.find(AliyunCredentials.aliyun_access_key_id) >= 0:
                        if len(line.split("=",1)) == 2:
                            key = line.split("=",1)[1].replace("\n", "")
                return key
            finally:
                f.close()

    def getUserSecret(self):
        credentials = self.aliyunConfigurePath+self.credentials
        if os.path.exists(credentials):
            f = open(credentials, 'r')
            secret = None
            try:
                while True:
                    line = f.readline()
                    if not line:
                        break
                    if line.find(AliyunCredentials.aliyun_access_key_secret) >= 0:
                        if len(line.split("=",1)) == 2:
                            secret = line.split("=",1)[1].replace("\n", "")
                return secret
            finally:
                f.close()

if __name__ == "__main__":
    handler = aliyunExtensionCliHandler()
    print handler.getUserRegion()
    print handler.getUserFormat()
    print handler.getUserKey()
    print handler.getUserSecret()
    print handler.getConfig()
# this function will show aliyun cli help
