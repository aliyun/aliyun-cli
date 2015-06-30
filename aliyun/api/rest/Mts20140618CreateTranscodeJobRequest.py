'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Mts20140618CreateTranscodeJobRequest(RestApi):
	def __init__(self,domain='mts.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.MediaId = None
		self.TemplateId = None
		self.Watermarks = None

	def getapiname(self):
		return 'mts.aliyuncs.com.CreateTranscodeJob.2014-06-18'
