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
import sys,json
import paramOptimize

class aliyunCliParser():
    def __init__(self):
        self.args = sys.argv[1:]
        pass

# this function find cli cmd
    def getCliCmd(self):
        if self.args.__len__() >= 1:
            return self.args[0].lower()

# this function find cli operation
    def getCliOperation(self):
        if self.args.__len__() >= 2:
            return self.args[1]

    def _getCommand(self):
        if self.args.__len__() >=1:
            return self.args[0]

    def _getOperations(self):
        operations = []
        i =1
        _len = self.args.__len__()
        if _len >=2:
            while i < _len:
                if  self.args[i].strip().find('--'):
                    operations.append(self.args[i])
                else:
                    break
                i =i+1
        if len(operations):
            return operations
        else :
            return None



    def _getKeyValues(self):
        keyValues = dict()
        len = self.args.__len__()

        # check is using ecs cli now?
        ecsFlag = False
        if "ecs" == str(self.getCliCmd()):
            ecsFlag = True

        emrFlag = False            
        if  "emr"  ==  str(self.getCliCmd()):
            emrFlag = True

        if len >= 2:
            current=1
            while current <len:
                #values = list()
                if self.args[current].strip().startswith('--'):
                    key=self.args[current].strip()
                    if ecsFlag:  # if using ecs cli, we need compate xx.yy.zz and xxyyzz in cli
                        key = key.replace('.', '')
                    start=current + 1
                    values=list()
                    while start <len and not self.args[start].strip().startswith('--'):
                        v = self.args[start].strip()
                        if emrFlag and   key.strip("--") in {'BootstrapActions', 'ClusterTypeLists', 'Columns', 'EcsOrders', 'EcsResetAutoRenewDos', 'ExecutionPlanIdLists', 'JobIdLists', 'OptionSoftWareLists', 'RenewEcsDos', 'StatusLists'} :
                            try:
                                v  = v.replace("'","\"")
                                v = json.loads(v)
                            except ValueError as e:
                                print("aliyuncli JSON argument parse failed. Detail message:")
                                print("\tcurrent argument:{0}".format(key))
                                print("\tdiagnostic message: {0} ".format(e.message))
                                sys.exit(-1)
                        values.append(v)
                        start=start+1
                    keyValues[key] = values
                    current=start
                else:
                    current=current+1
        paramOptimize._paramOptimize(keyValues)
        return keyValues

# this function find cli key:values , notice here is values , we need consider multiple values case
# --args is key, and if no -- is value
    def getCliKeyValues(self):
        keyValues = dict()
        len = self.args.__len__()
        if len >= 3:
            left_index = 2
            if self.args[1].find("--") >= 0:
                left_index = 1
            for index in range(left_index, len):
                currentValue = self.args[index]
                if currentValue.find('--') >= 0 : # this is command
                    index = index+1 # check next args
                    values = list()
                    while index < len and self.args[index].find('--') < 0:
                        values.append(self.args[index])
                        index = index + 1
                    keyValues[currentValue] = values
        return keyValues

# this function will find the temp key and secret if user input the --key and --value
    def getTempKeyAndSecret(self):
        keyValues = dict()
        len = self.args.__len__()
        keystr = "--AccessKeyId"
        secretstr = "--AccessKeySecret"
        _key = None
        _secret = None
        if len >= 3:
            for index in range(2, len):
                currentValue = self.args[index]
                if currentValue.find('--') >= 0 : # this is command
                    index = index+1 # check next args
                    values = list()
                    while index < len and self.args[index].find('--') < 0:
                        values.append(self.args[index])
                        index = index + 1
                    keyValues[currentValue] = values
        if keyValues.has_key(keystr) and keyValues[keystr].__len__() > 0:
            _key = keyValues[keystr][0]
        if keyValues.has_key(secretstr) and keyValues[secretstr].__len__() > 0:
            _secret = keyValues[secretstr][0]
        #print "accesskeyid: ", _key , "accesskeysecret: ",_secret
        return _key, _secret



# this function will give all extension command defined by us
    def getAllExtensionCommands(self):
        cmds = list()
        cmds = ['help', '-h', '--help', ]
        return cmds

# this function will filter all key and values which is in openApi
    def getOpenApiKeyValues(self, map):
        keys = map.keys()
        newMap = dict()
        for key in keys:
            value = map.get(key)
            key = key.replace('--', '')
            newMap[key] = value
        return newMap
    

    def _getOpenApiKeyValues(self, map):
        keys = map.keys()
        newMap = dict()
        for key in keys:
            value = map.get(key)
            key = key.replace('--', '')
            newMap[key] = value
        return newMap

# this function will filter all key and values which is in extension command

# this function will filter all key and values which is in extension command
    def getExtensionKeyValues(self, map):
        pass

# this function will return output format from key values
    def getOutPutFormat(self, map):
        keys = map.keys()
        for key in keys:
            if key == '--output' :
                return map.get(key)
        return None

# this function will return whether to use HTTPS request
    def getSecureChoice(self, map):
        keys = map.keys()
        for key in keys:
            if key == '--secure' :
                return True 
        return False
    


