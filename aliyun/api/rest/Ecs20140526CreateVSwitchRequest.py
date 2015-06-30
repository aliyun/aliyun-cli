'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Ecs20140526CreateVSwitchRequest(RestApi):
	def __init__(self,domain='ecs.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.CidrBlock = None
		self.ClientToken = None
		self.Description = None
		self.VSwitchName = None
		self.VpcId = None
		self.ZoneId = None

	def getapiname(self):
		return 'ecs.aliyuncs.com.CreateVSwitch.2014-05-26'
