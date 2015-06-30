'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Mts20140618SearchTranscodeJobListRequest(RestApi):
	def __init__(self,domain='mts.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.JobState = None
		self.LowerBoundOfJobCreatedTime = None
		self.PageNumber = None
		self.PageSize = None
		self.UpperBoundOfJobCreatedTime = None

	def getapiname(self):
		return 'mts.aliyuncs.com.SearchTranscodeJobList.2014-06-18'
