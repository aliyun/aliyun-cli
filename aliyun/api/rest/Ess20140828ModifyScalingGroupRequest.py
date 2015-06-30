'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Ess20140828ModifyScalingGroupRequest(RestApi):
	def __init__(self,domain='ess.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.ActiveScalingConfigurationId = None
		self.DefaultCooldown = None
		self.MaxSize = None
		self.MinSize = None
		self.RemovalPolicy_1 = None
		self.RemovalPolicy_2 = None
		self.ScalingGroupId = None
		self.ScalingGroupName = None

	def getapiname(self):
		return 'ess.aliyuncs.com.ModifyScalingGroup.2014-08-28'

	def getTranslateParas(self):
		return {'RemovalPolicy_1':'RemovalPolicy.1','RemovalPolicy_2':'RemovalPolicy.2'}
