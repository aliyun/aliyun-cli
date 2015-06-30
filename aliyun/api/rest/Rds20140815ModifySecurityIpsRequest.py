'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Rds20140815ModifySecurityIpsRequest(RestApi):
	def __init__(self,domain='rds.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.DBInstanceId = None
		self.SecurityIps = None

	def getapiname(self):
		return 'rds.aliyuncs.com.ModifySecurityIps.2014-08-15'
