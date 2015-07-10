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

import aliyunCliConfiugre
import urllib2
import re
import os
import platform
class updateFileHandler(object):
    def __init__(self):
        self.home = ".aliyuncli"
        self.update = "update"
        self.tag = "/"
        if platform.system() == "Windows":
            self.tag = "\\"
        self.aliyuncliUpdatePath = self._findHomePath()+self.tag+self.home+self.tag
        self.globalConfigure = aliyunCliConfiugre.configure()
    def _getValueFromFile(self,filename,key):
        if os.path.isfile(filename):
            with open(filename, 'r') as f:
                contents = f.readlines()
                value = self._getValueFromContents(key,contents)
                return value
        else:
            return None
    def _getValueFromContents(self,key,contents):
        value = None
        if contents is None:
            return None
        else:
            for i in range(len(contents)):
                line = contents[i]
                key = self._getKey(line)
                if key  is not None:
                    if key.strip().lower() == 'ignore':
                        value = self._getValue(line)
        return value

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

    def _findHomePath(self):
        homePath = ""
        if platform.system() == "Windows":
            homePath = os.environ['HOMEPATH']
            pass
        else:
            homePath = os.environ['HOME']
            pass
        return homePath
    def _getUpdateFileName(self):
        filename = None
        filename = self.aliyuncliUpdatePath+ self.update
        return filename
class aliyunCliUpgradeHandler():
    def __init__(self):
        self.configure = aliyunCliConfiugre.configure()
        self.latest_update_time = self._getLatestTimeFromServer()
        self._updateFileHandler = updateFileHandler()

# this function give the upgrade info
    def checkForUpgrade(self):
        filename = self._updateFileHandler._getUpdateFileName()
        ignore = 'ignore'
        ignoreValue = self._updateFileHandler._getValueFromFile(filename,ignore)
        ignore_info = 'no'
        if ignoreValue is not None:
            ignore_info = ignoreValue
        need_notify = True
        latest_update_time = self.getLatestTimeFromServer().replace("\n","")
        if ignore_info.find("yes")>=0 and len(ignore_info.split("_"))==2: #should be yes_20150426 format
            if latest_update_time == ignore_info.split("_")[1]:
                need_notify = False
        if need_notify:
            isNewVersion, url = self.isNewVersionReady()
            if isNewVersion:
                return True, url
            else:
                return False, ""
        else:
            return False, ""

# this function handle user choice when notify the new version is comming
    def handleUserChoice(self, choice):
        updatePath = self._updateFileHandler._getUpdateFileName()
        latest_update_time = self.getLatestTimeFromServer().replace("\n","")
        _ignore = "no"
        if choice in ["no", "NO", "n", "N"]:
            _ignore = "yes_"+latest_update_time
        self._createFile(updatePath)
        fd = open(updatePath, 'w')
        try:
            configtxt ="ignore="+_ignore+"\n"
            fd.write(configtxt)
        finally:
            fd.close()

    def _createFile(self,filename):
        namePath = os.path.split(filename)[0]
        if not os.path.isdir(namePath):
            os.makedirs(namePath)
            with os.fdopen(os.open(filename, os.O_WRONLY | os.O_CREAT, 0o600), 'w'):
                pass


# this function checks if there is new version
    def isNewVersionReady(self):
        latest_update_time = self.getLatestTimeFromServer()
        current_update_time = self.getUpdateTime()
        # no matter what error happened , we should ignore it and keep the cli tool can run normally
        try:
            if int(latest_update_time) > int(current_update_time):
                return True, self.configure.update_url
            else:
                return False, ""
        except Exception as e:
            return False, ""

# this function will give the current version
    def getUpdateTime(self):
        return self.configure.update_time

# this functino will get the latest version
    def getLatestTimeFromServer(self):
        return self.latest_update_time

# this functino will get the latest version
    def _getLatestTimeFromServer(self):
        try:
            f = urllib2.urlopen(self.configure.server_url,data=None,timeout=5)
            s = f.read()
            return s
        except Exception as e:
            return ""



if __name__ == "__main__":
    upgradeHandler = aliyunCliUpgradeHandler()
    # print upgradeHandler.getLatestTimeFromServer()
    # flag, url = upgradeHandler.isNewVersionReady()
    # if flag:
    #     print url
    # else:
    #     print "current version is latest one"
    # print "final test:"
    print upgradeHandler.checkForUpgrade()
    print upgradeHandler.handleUserChoice("N")
