__author__ = 'xixi.xxx'
import os,sys
parentdir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0,parentdir)
import aliyunExtensionCliHandler
class ProfileCmd:
    useProfile = 'useProfile'
    addProfile = 'addProfile'
    name = '--name'
class ProfileHandler:
    def __init__(self):
        self.extensionCliHandler = aliyunExtensionCliHandler.aliyunExtensionCliHandler()

    def getProfileHandlerCmd(self):
        return [ProfileCmd.useProfile, ProfileCmd.addProfile]

    def getProfileHandlerOptions(self):
        return ['--name']

    def handleProfileCmd(self, cmd, keyValues):
        if cmd.lower() == ProfileCmd.useProfile.lower(): # confirm command is right
            #check --name is valid
            if keyValues.has_key(ProfileCmd.name) and len(keyValues[ProfileCmd.name]) > 0:
                _value = keyValues[ProfileCmd.name][0] # use the first value
                self.extensionCliHandler.setUserProfile(_value)
            else:
                print "Do your forget profile name? please use \'--name\' and add the profile name."
        else:
            print "[", cmd, "] is not right, do you mean "+ProfileCmd.useProfile+" ?"

    def addProfileCmd(self, cmd, keyValues):
        userKey = ''
        userSecret = ''
        newProfileName = ''
        if cmd.lower() == ProfileCmd.addProfile.lower(): # confirm command is right
            #check --name is valid
            if keyValues.has_key(ProfileCmd.name) and len(keyValues[ProfileCmd.name]) > 0:
                _value = keyValues[ProfileCmd.name][0] # check the first value
                # only input key and secret
                newProfileName = _value
            else:
                # need input profilename key and value
                newProfileName = raw_input("profile name [None]: ")
            userKey = raw_input("Aliyun Access Key ID [None]: ")
            userSecret = raw_input("Aliyun Secret Access Key [None]: ")
            _credentialsPath = os.path.join(self.extensionCliHandler.aliyunConfigurePath,self.extensionCliHandler.credentials)
            if os.path.exists(_credentialsPath):
                f = open(_credentialsPath, 'a')
                try:
                    content = "\n["+newProfileName+"]\naliyun_access_key_id="+userKey+"\naliyun_secret_access_key="+userSecret+"\n"
                    f.write(content)
                finally:
                    f.close()
        else:
            print "[", cmd, "] is not right, do you mean "+ProfileCmd.addProfile+" ?"


if __name__ == "__main__":
    handler = ProfileHandler()
    handler.handleProfileCmd("useprofile", {'--name':["profile444"]})
    print handler.extensionCliHandler.getUserProfile()
    handler.addProfileCmd("addProfile", {})
    handler.addProfileCmd("addProfile", {'--name':["profile2222"]})