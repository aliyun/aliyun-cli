'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Yundun20140924SetSLBDdosSettingRequest(RestApi):
	def __init__(self,domain='yundun.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.bps = None
		self.instanceId = None
		self.isLayer7 = None
		self.pps = None
		self.qps = None
		self.sipConn = None
		self.sipNew = None

	def getapiname(self):
		return 'yundun.aliyuncs.com.setSLBDdosSetting.2014-09-24'
