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
#/usr/bin/env python
#-*- coding:utf-8 -*-

import os,sys
import aliyunCliParser
import platform


OSS_CREDS_FILENAME = "%s/.aliyuncli/osscredentials" % os.path.expanduser('~')
OSS_CONFIG_SECTION = 'OSSCredentials'

OAS_CREDS_FILENAME = "%s/.aliyuncli/oascredentials" % os.path.expanduser('~')
OAS_CONFIG_SECTION = 'OASCredentials'
def handleConfigure(cmd,operation):
    if not hasConfigureOperation(cmd,operation):
        handler = ConfigureCommand()
        handler._run_main()
    elif operation[0].lower() == 'list':
        handler = ConfigureListCommand()
        handler._run_main()
    elif operation[0].lower()== 'get':
        handler =ConfigureGetCommand()
        handler._run_main()
    elif operation[0].lower() == 'set':
        handler = ConfigureSetCommand()
        handler._run_main()


def hasConfigureOperation(cmd,operation):
    if  operation is not None:
        return True
    else :
        return False

class AliyunCredentials:
    aliyun_access_key_id = "aliyun_access_key_id"
    aliyun_access_key_secret = "aliyun_access_key_secret"

_WRITE_TO_CREDS_FILE = ['aliyun_access_key_id', 'aliyun_access_key_secret']
_WRITE_TO_CONFIG_FILE = ['region','output']




class AliyunConfig(object):
    def __init__(self,configWriter = None):
        if configWriter is None:
            configWriter = ConfigFileWriter()
        self._configWriter = configWriter
        self.home = ".aliyuncli"
        self.configure = "configure"
        self.credentials = "credentials"
        self.aliyunConfigurePath = os.path.join(self.findConfigureFilePath(),self.home)
        self.parser = aliyunCliParser.aliyunCliParser()

    def getConfig(self,profilename = None):
        if profilename is None:
            profilename = 'default'
        config = {}
        _configFileName = self.getConfigFileName()
        _credsFileName = self.getCredsFileName()
        self._getConfigFromFile(config,profilename,_configFileName)
        self._getConfigFromFile(config,profilename,_credsFileName)
        return config

    def _getConfigFromFile(self,config,profile,filename):
        if os.path.isfile(filename):
            with open(filename, 'r') as f:
                contents = f.readlines()
                if profile == 'default':
                    _sectionName = '%s'%profile
                else:
                    _sectionName = 'profile %s'%profile
                if self._configWriter.hasSectionName(_sectionName,contents)[0]:
                    sectionStart = self._configWriter.hasSectionName(_sectionName,contents)[1]
                    j = sectionStart
                    sectionEnd = self._configWriter._getSectionEnd(j,contents)
                    while j < sectionEnd:
                        line = contents[j]
                        if line.strip():
                            key = self._configWriter._getKey(line)
                            config[key] = self._configWriter._getValue(line)
                        j = j+1
                else:
                    pass
        else:
            pass

    def getConfigFileName(self):
        configFileName = os.path.join(self.aliyunConfigurePath,self.configure)
        return configFileName

    def getCredsFileName(self):
        credsFileName = os.path.join(self.aliyunConfigurePath,self.credentials)
        return credsFileName


    def findConfigureFilePath(self):
        homePath = ""
        if platform.system() == "Windows":
            homePath = os.environ['HOMEPATH']
            pass
        else:
            homePath = os.environ['HOME']
            pass
        return homePath


    def _getUserRegion(self):
        _keyValues = self.parser._getKeyValues()
        _profileName = _keyValues.get('--profile')
        if _profileName is not None and len(_profileName)>0:
            _profileName = _profileName[0]
        _config = self.getConfig(_profileName)
        _region = _config.get('region')
        return _region

    def _getUserFormat(self):
        _keyValues = self.parser._getKeyValues()
        _profileName = _keyValues.get('--profile')
        if _profileName is not None and len(_profileName)>0:
            _profileName = _profileName[0]
        _config = self.getConfig(_profileName)
        _format = _config.get('output')
        return _format


    def _getUserKey(self):
        _keyValues = self.parser._getKeyValues()
        _profileName = _keyValues.get('--profile')
        if _profileName is not None and len(_profileName)>0:
            _profileName = _profileName[0]
        _config = self.getConfig(_profileName)
        _userKey = _config.get(AliyunCredentials.aliyun_access_key_id)
        return _userKey

    def _getUserSecret(self):
        _keyValues = self.parser._getKeyValues()
        _profileName = _keyValues.get('--profile')
        if _profileName is not None and len(_profileName)>0:
            _profileName = _profileName[0]
        _config = self.getConfig(_profileName)
        _secretId = _config.get(AliyunCredentials.aliyun_access_key_secret)
        return _secretId

    def _getPort(self):
        _keyValues = self.parser._getKeyValues()
        _profileName = _keyValues.get('--profile')
        if _profileName is not None and len(_profileName)>0:
            _profileName = _profileName[0]
        _config = self.getConfig(_profileName)
        _port = _config.get('port')
        return _port

class InteractivePrompter(object):
    def __init__(self):
        pass
    def get_value(self, current_value, config_name, prompt_text=''):
        if config_name in ('aliyun_access_key_id', 'aliyun_access_key_secret'):
            if current_value != '' and not current_value is None:
                current_value = _mask_value(current_value)
        response = raw_input("%s [%s]: " % (prompt_text, current_value))
        if response is '':
            response = None
        return response

def _mask_value(current_value):
    if current_value is None:
        return 'None'
    else:
        if len(current_value) <4:
            count = 20 - len(current_value)
            return ('*'* count) + current_value[-4:]
        else:
            return ('*' * 16) + current_value[-4:]

def _createFile(filename):
    namePath = os.path.split(filename)[0]
    if not os.path.isdir(namePath):
        os.makedirs(namePath)
        with os.fdopen(os.open(filename,
                               os.O_WRONLY | os.O_CREAT, 0o600), 'w'):
            pass

class ConfigFileWriter(object):
    def __init__(self):
        pass

    def _updateConfig(self,new_values,config_filename):
        sectionName  = new_values.pop('__section__','default')
        if not os.path.isfile(config_filename):
            _createFile(config_filename)
            self._insertNewSection(sectionName,new_values,config_filename)
            return
        with open(config_filename, 'r') as f:
            contents = f.readlines()
        try:
            if self.hasSectionName(sectionName,contents)[0]:
                self._updateSectionContents(sectionName,contents, new_values)
                with open(config_filename, 'w') as f:
                    f.write(''.join(contents))
            else:
                self._insertNewSection(sectionName,new_values,config_filename)
        finally:
            f.close()



    def hasSectionName(self,sectionName,contents):
        result = False
        start = -1
        _sectionName = '['+sectionName+']'
        for i in range(len(contents)):
            line = contents[i]
            if line.strip():
                if line.strip().startswith(('#', ';')):
                    continue
                if line.strip().find(_sectionName)  >=0:
                    result = True
                    start = i
                    break
        return result,start

    def _updateSectionContents(self,sectionName,contents,new_values):
        new_values = new_values.copy()
        sectionStart = self.hasSectionName(sectionName,contents)[1]
        j = sectionStart
        sectionEnd = self._getSectionEnd(j,contents)
        while j < sectionEnd:
            line = contents[j]
            key = self._getKey(line)
            if key in new_values:
                new_value = new_values[key]
                new_line =  '%s = %s\n' % (key, new_value)
                contents[j] = new_line
                del new_values[key]
            j = j +1
        if new_values:
            if not contents[-1].endswith('\n'):
                contents.append('\n')
            self._insertNewValues(sectionEnd,contents,new_values)

    def _getSectionEnd(self,num,contents):
        k = len(contents)
        num = num +1
        while num <len(contents):
            line = contents[num]
            if line.strip():
                if line.strip().startswith(('#', ';')):
                    continue
                if  line.find('[') >=0 and line.find(']') :
                    return num
            num = num +1
        return k

    def _insertNewValues(self,num,contents,keyValues):
        new_contents = []
        for key, value in list(keyValues.items()):
            new_contents.append('%s = %s\n' % ( key, value))
            del keyValues[key]
        contents.insert(num, ''.join(new_contents))


    def _insertNewSection(self,sectionName,new_values,config_filename,num =0):
        with open(config_filename, 'a') as f:
            f.write('[%s]\n' % sectionName)
            contents = []
            self._insertNewValues(num ,contents,new_values)
            f.write(''.join(contents))


    def _getKey(self,line):
        key = None
        if line is not None and  line.strip().find('=') >0 :
            key = line.split("=",1)[0].strip()
        return key
    def _getValue(self,line):
        value = None
        if line is not None and  line.strip().find('=') >0 :
            value = line.split("=",1)[1].strip()
        return value
    def _getValueInSlice(self,start,end,key,contents):
        value = None
        while start <end:
            line = contents[start]
            start = start+1
            if key ==self._getKey(line):
                value = self._getValue(line)
                return value
        return value





class ConfigureSetCommand(object):
    def __init__(self,aliyunConfig= None,configWriter =None):
        if configWriter is None:
            configWriter = ConfigFileWriter()
        self._configWriter = configWriter
        if aliyunConfig is None:
            self.aliyunConfig = AliyunConfig()

    def _run_main(self):
        keyValues = self.aliyunConfig.parser._getKeyValues()
        _profilenameList = keyValues.get('--profile')
        _profilename = None
        if _profilenameList is not None:
            if len(_profilenameList) > 0:
                _profilename = _profilenameList[0]
        if _profilename is  not None:
            del keyValues['--profile']
        newValues = self.aliyunConfig.parser._getOpenApiKeyValues(keyValues)
        for key in newValues.keys():
            if newValues[key] is not None:
                if len(newValues[key]) > 0:
                    newValues[key] = newValues[key][0]
            else:
                del newValues[key]
        config_filename = self.aliyunConfig.getConfigFileName()
        if newValues:
            self._writeCredsToFile(newValues,_profilename)
            if _profilename is not None :
                newValues['__section__']=('profile %s' % _profilename)
            self._configWriter._updateConfig(newValues,config_filename)

    def _writeCredsToFile(self,new_values,profilename):
        credential_file_values = {}
        if 'AccessKeyId' in new_values:
            credential_file_values['aliyun_access_key_id'] = new_values.pop(
                'AccessKeyId')
        if 'AccessKeySecret' in new_values:
            credential_file_values['aliyun_access_key_secret'] = new_values.pop(
                'AccessKeySecret')
        creds_filename = self.aliyunConfig.getCredsFileName()
        if credential_file_values:
            if profilename is not None:
                credential_file_values['__section__'] = ('profile %s' % profilename)
            self._configWriter._updateConfig(credential_file_values,creds_filename)

class ConfigValue(object):

    def __init__(self, value, config_type, config_loc):
        self.value = value
        self.config_type = config_type
        self.config_loc = config_loc

    def mask_value(self):
        if self.value is None:
            return
        self.value = _mask_value(self.value)

class ConfigureListCommand(object):
    def __init__(self,aliyunConfig= None, stream=None,configWriter =None):
        if stream is None :
            self._stream = sys.stdout
        if aliyunConfig is None:
            self.aliyunConfig = AliyunConfig()
        if configWriter is None:
            configWriter = ConfigFileWriter()
        self._configWriter = configWriter
    def _run_main(self):
        self._displayConfigValue(ConfigValue('Value', 'Type', 'Location'),'Name')
        self._displayConfigValue(ConfigValue('-----', '----', '--------'),'----')
        keyValues = self.aliyunConfig.parser._getKeyValues()
        _profilenameList = keyValues.get('--profile')
        _profilename = None
        if _profilenameList is not None:
            if len(_profilenameList) > 0:
                _profilename = _profilenameList[0]
        sectionName = 'default'
        if _profilename is  not None:
            sectionName = ('profile %s' % _profilename)
        profile = ConfigValue(_profilename, 'None','None')
        self._displayConfigValue(profile,'Profile')
        config_filename = self.aliyunConfig.getConfigFileName()
        creds_filename = self.aliyunConfig.getCredsFileName()
        access_key, secret_key = self._lookup_credentials(sectionName,creds_filename)
        region,output = self._lookup_config(sectionName,config_filename)
        self._displayConfigValue(access_key,'Access_Key')
        self._displayConfigValue(secret_key,'Secret_Key')
        self._displayConfigValue(region,'Region')
        self._displayConfigValue(output,'Output')

    def _lookup_credentials(self,sectionName,filename):
        access_key = _WRITE_TO_CREDS_FILE[0]
        secret_key = _WRITE_TO_CREDS_FILE[1]
        access_type = None
        secret_type = None
        access_key_value = self.getConfigValue(access_key,sectionName,filename)
        secret_key_value = self.getConfigValue(secret_key,sectionName,filename)
        if access_key_value is not None:
            access_type = 'credentials'
        if secret_key_value is not None:
            secret_type = 'credentials'
        if access_key_value is None and secret_key_value is None:
            filename = None
        access_key_value = _mask_value(access_key_value)
        secret_key_value = _mask_value(secret_key_value)
        accessConfigValue =ConfigValue(access_key_value,access_type,filename)
        secretConfigValue =ConfigValue(secret_key_value,secret_type,filename)
        return accessConfigValue,secretConfigValue

    def _lookup_config(self,sectionName,filename):
        region_key = 'region'
        output_key = 'output'
        region_type = None
        output_type = None
        region_value = self.getConfigValue(region_key,sectionName,filename)
        output_value = self.getConfigValue(output_key,sectionName,filename)
        if region_value is not None:
            region_type ='configure'
        if output_value is not None:
            output_type = 'configure'
        if region_value is None and output_value is None:
            filename = None
        regionConfigValue = ConfigValue(region_value,region_type,filename)
        outputConfigValue = ConfigValue(output_value,output_type,filename)
        return regionConfigValue,outputConfigValue


    def getConfigValue(self,key,sectionName,filename):
        _value = None
        with open(filename, 'r') as f:
            contents = f.readlines()
        try:
            if self._configWriter.hasSectionName(sectionName,contents)[0]:
                j = self._configWriter.hasSectionName(sectionName,contents)[1]
                sectionEnd = self._configWriter._getSectionEnd(j,contents)
                while j < sectionEnd:
                    line = contents[j]
                    _key = self._configWriter._getKey(line)
                    if key == _key:
                        _value = self._configWriter._getValue(line)
                    j = j +1
        finally:
            f.close()
        return _value



    def _displayConfigValue(self,configValue,configName):
        self._stream.write('%10s %20s %15s    %20s\n' % (
            configName, configValue.value, configValue.config_type,
            configValue.config_loc))
        pass


class ConfigureCommand(object):
    def __init__(self,aliyunConfig= None,prompter =None,configWriter =None):
        if prompter is None:
            prompter = InteractivePrompter()
        self._prompter = prompter
        if configWriter is None:
            configWriter = ConfigFileWriter()
        self._configWriter = configWriter
        if aliyunConfig is None:
            self.aliyunConfig = AliyunConfig()

    name2prompt = [
        ('aliyun_access_key_id', "Aliyun Access Key ID"),
        ('aliyun_access_key_secret', "Aliyun Access Key Secret"),
        ('region', "Default Region Id"),
        ('output', "Default output format"),
    ]

    def _run_main(self):
        new_values = {}
        all_values = {}
        keyValues = self.aliyunConfig.parser._getKeyValues()
        _profilename = None
        _profilenameList = keyValues.get('--profile')
        if _profilenameList is not None:
            if len(_profilenameList) > 0:
                _profilename = _profilenameList[0]
        config = self.aliyunConfig.getConfig(_profilename)
        for name, prompt in self.name2prompt:
            value = config.get(name)
            new_value = self._prompter.get_value(value, name,prompt)
            all_values[name] = new_value
            if new_value is not None and new_value != value:
                new_values[name] = new_value
        config_filename = self.aliyunConfig.getConfigFileName()
        if all_values:
            self._writeCredsToOssFile(all_values)
            self._writeCredsToOasFile(all_values)
        if new_values:
            self._writeCredsToFile(new_values,_profilename)
            if _profilename is not None :
                new_values['__section__']=('profile %s' % _profilename)
            self._configWriter._updateConfig(new_values,config_filename)

    def _writeCredsToFile(self,new_values,profilename):
        credential_file_values = {}
        if 'aliyun_access_key_id' in new_values:
            credential_file_values['aliyun_access_key_id'] = new_values.pop(
                'aliyun_access_key_id')
        if 'aliyun_access_key_secret' in new_values:
            credential_file_values['aliyun_access_key_secret'] = new_values.pop(
                'aliyun_access_key_secret')
        creds_filename = self.aliyunConfig.getCredsFileName()
        if credential_file_values:
            if profilename is not None:
                credential_file_values['__section__'] = ('profile %s' % profilename)
            self._configWriter._updateConfig(credential_file_values,creds_filename)


    def _writeCredsToOssFile(self,new_values,profilename = OSS_CONFIG_SECTION):
        credential_file_values = {}
        if 'aliyun_access_key_id' in new_values:
            credential_file_values['accessid'] = new_values['aliyun_access_key_id']
        if 'aliyun_access_key_secret' in new_values:
            credential_file_values['accesskey'] = new_values['aliyun_access_key_secret']
        ossCredsFilename = OSS_CREDS_FILENAME
        if credential_file_values:
            if profilename is not None:
                credential_file_values['__section__'] = ('%s' % profilename)
            self._configWriter._updateConfig(credential_file_values,ossCredsFilename)

    def _writeCredsToOasFile(self,new_values,profilename = OAS_CONFIG_SECTION):
        credential_file_values = {}
        if 'aliyun_access_key_id' in new_values:
            credential_file_values['accessid'] = new_values['aliyun_access_key_id']
        if 'aliyun_access_key_secret' in new_values:
            credential_file_values['accesskey'] = new_values['aliyun_access_key_secret']
        oasCredsFilename = OAS_CREDS_FILENAME
        if credential_file_values:
            if profilename is not None:
                credential_file_values['__section__'] = ('%s' % profilename)
            self._configWriter._updateConfig(credential_file_values,oasCredsFilename)






class ConfigureGetCommand(object):
    def __init__(self,aliyunConfig= None,configWriter =None):
        if configWriter is None:
            configWriter = ConfigFileWriter()
        self._configWriter = configWriter
        if aliyunConfig is None:
            aliyunConfig = AliyunConfig()
        self.aliyunConfig = aliyunConfig

    def _run_main(self):
        keyValues = self.aliyunConfig.parser._getKeyValues()
        _profilenameList = keyValues.get('--profile')
        _profilename = None
        if _profilenameList is not None:
            if len(_profilenameList) > 0:
                _profilename = _profilenameList[0]
        operations = self.aliyunConfig.parser._getOperations()
        self._getKeysFromSection(_profilename,operations)

    def _getKeysFromSection(self,profilename,operations):
        if len(operations) >= 2:
            for i in range(1,len(operations)):
                operation = operations[i].strip()
                self._getKeyFromSection(profilename,operation)
        else:
            print 'The correct usage:aliyuncli configure get key --profile profilename'
            return

    def _getKeyFromSection(self,profilename,key):
        sectionName = 'default'
        if profilename is not None:
            sectionName = ('profile %s' % profilename)
        config_filename = self.aliyunConfig.getConfigFileName()
        creds_filename = self.aliyunConfig.getCredsFileName()
        if key in _WRITE_TO_CREDS_FILE :
            self._getKeyFromFile(creds_filename,sectionName,key)
        elif key in _WRITE_TO_CONFIG_FILE :
            self._getKeyFromFile(config_filename,sectionName,key)
        else:
            print key,'=','None'
    def _getKeyFromFile(self,filename,section,key):
        if  os.path.isfile(filename):
            with open(filename, 'r') as f:
                contents = f.readlines()
            if self._configWriter.hasSectionName(section,contents)[0]:
                start =  self._configWriter.hasSectionName(section,contents)[1]
                end = self._configWriter._getSectionEnd(start,contents)
                value = self._configWriter._getValueInSlice(start,end,key,contents)
                print key,'=',value
        else:
            print key,'=None'




if __name__ == "__main__":
    pass
