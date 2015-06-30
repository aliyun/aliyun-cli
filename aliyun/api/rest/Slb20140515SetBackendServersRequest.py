'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Slb20140515SetBackendServersRequest(RestApi):
	def __init__(self,domain='slb.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.BackendServers = None
		self.LoadBalancerId = None

	def getapiname(self):
		return 'slb.aliyuncs.com.SetBackendServers.2014-05-15'
