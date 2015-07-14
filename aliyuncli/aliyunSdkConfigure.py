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
#!/usr/bin/env python
#-*- coding:utf-8 -*-
import os,platform,sys
import aliyunCliParser

def handleSdkVersion(cmd,operation,version):
    handler=AliyunSdkConfigure()
    if  operation.lower() == 'configversion':
        handler.sdkConfigure(cmd,operation)
    elif operation.lower()== 'showversions':
        handler.showVersions(cmd,operation)


class AliyunSdkConfigure(object):
    def __init__(self):
        self.home = ".aliyuncli"
        self.sdkFile = "sdk_version"
        self.aliyunConfigurePath = os.path.join(self.findConfigureFilePath(),self.home)
        self.fileName=os.path.join(self.aliyunConfigurePath,self.sdkFile)
        self.parser = aliyunCliParser.aliyunCliParser()

    def sdkConfigure(self,cmd,operation):
        keyValues = self.parser._getKeyValues()
        if keyValues.has_key('--version') and len(keyValues['--version']) > 0:
            version=keyValues['--version'][0]
            filename=self.fileName
            self.writeCmdVersionToFile(cmd,version,filename)
        else:
            print "A argument is needed! please use \'--version\' and add the sdk version."
            return
    def showVersions(self,cmd,operation,stream=None):
        configureVersion='(not configure)'
        if stream is None:
            stream=sys.stdout
        getVersion=self.getCmdVersionFromFile(cmd,self.fileName)
        if  getVersion is not None:
            configureVersion=getVersion
        import aliyunOpenApiData
        apiHandler =aliyunOpenApiData.aliyunOpenApiDataHandler()
        versions =apiHandler.getAllVersionsByCmdName(cmd)
        stream.write('%s%s\n' % ('* ',configureVersion))
        for version in versions:
            stream.write('  %s\n' % (version))




    def findConfigureFilePath(self):
        homePath = ""
        if platform.system() == "Windows":
            homePath = os.environ['HOMEPATH']
            pass
        else:
            homePath = os.environ['HOME']
            pass
        return homePath

    def getCmdVersionFromFile(self,cmd,filename):
        if not os.path.isfile(filename):
            return None
        with open(filename, 'r') as f:
            contents = f.readlines()
        try:
            if self.hasCmdName(cmd,contents):
                keyValues=self._getKeyValues(contents)
                version=keyValues.get(cmd)
                return version
            else:
                return None
        finally:
            f.close()

    def updateCmdVersion(self,cmd,version,filename):
        if not os.path.isfile(filename):
            self._createFile(filename)
            self.insertCmdVersion(cmd,version,filename)
            return
        with open(filename, 'r') as f:
            contents = f.readlines()
            try:
                if self.hasCmdName(cmd,contents):
                    self._updateContents(cmd,version,contents)
                    with open(filename, 'w') as f:
                        f.write(''.join(contents))
                else:
                    self.insertCmdVersion(cmd,version,filename)
            finally:
                f.close()

    def insertCmdVersion(self,cmd,version,filename):
        with open(filename, 'a') as f:
            contents = []
            self._insertNewCmdVersion(cmd,version,contents)
            f.write(''.join(contents))

    def writeCmdVersionToFile(self,cmd,version,filename):
        if cmd is not None and version is not None :
            if filename is not None :
                self.updateCmdVersion(cmd,version,filename)

    def hasCmdName(self,cmd,contents):
        keys=self._getKeys(contents)
        if cmd in keys:
            return True
        else:
            return False

    def _updateContents(self,cmd,version,contents):
        for j in range(len(contents)):
            line = contents[j]
            key = self._getKey(line)
            if cmd ==key:
                new_line = '%s=%s\n' %(cmd,version)
                contents[j] = new_line
                break

    def _insertNewCmdVersion(self,cmd,version,contents):
        new_contents = []
        new_contents.append('%s=%s\n' %(cmd,version))
        contents.append(''.join(new_contents))

    def _getKeys(self,contents):
        keys=[]
        for j in range(len(contents)):
            line = contents[j]
            key = self._getKey(line)
            keys.append(key)
        return keys

    def _getKey(self,line):
        key = None
        if line is not None and  line.strip().find('=') >0 :
            key = line.split("=",1)[0].strip()
        return key

    def _getKeyValues(self,contents):
        keyValues={}
        for j in range(len(contents)):
            line = contents[j]
            if line is not None and  line.strip().find('=') >0 :
                key = line.split("=",1)[0].strip()
                value = line.split("=",1)[1].strip()
                keyValues[key]=value
        return keyValues

    def _createFile(self,filename):
        namePath = os.path.split(filename)[0]
        if not os.path.isdir(namePath):
            os.makedirs(namePath)
            with os.fdopen(os.open(filename,
                                   os.O_WRONLY | os.O_CREAT, 0o600), 'w'):
                pass
