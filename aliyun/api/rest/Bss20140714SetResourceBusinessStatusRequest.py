'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Bss20140714SetResourceBusinessStatusRequest(RestApi):
	def __init__(self,domain='bss.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.BusinessStatus = None
		self.ResourceId = None
		self.ResourceType = None

	def getapiname(self):
		return 'bss.aliyuncs.com.SetResourceBusinessStatus.2014-07-14'
