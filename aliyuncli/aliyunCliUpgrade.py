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
import aliyunExtensionCliHandler
class aliyunCliUpgradeHandler():
    def __init__(self):
        self.configure = aliyunCliConfiugre.configure()
        self.extensionHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()
        self.latest_update_time = self._getLatestTimeFromServer()

# this function give the upgrade info
    def checkForUpgrade(self):
        ignore_info = self.extensionHandler.getIgnoreValues()
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
        configurePath = self.extensionHandler.aliyunConfigurePath+self.extensionHandler.configure
        latest_update_time = self.getLatestTimeFromServer().replace("\n","")
        _format = self.extensionHandler.getUserFormat()
        _region = self.extensionHandler.getUserRegion()
        _ignore = "no"
        if choice in ["no", "NO", "n", "N"]:
            _ignore = "yes_"+latest_update_time
        fd = open(configurePath, 'w')
        try:
            configtxt = "[default]\nregion="+_region+"\noutput="+_format+"\n"
            configtxt = configtxt + "ignore="+_ignore+"\n"
            fd.write(configtxt)
        finally:
            fd.close()


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
