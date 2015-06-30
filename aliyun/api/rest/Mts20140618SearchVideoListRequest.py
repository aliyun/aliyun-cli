'''
Created by auto_sdk on 2014-11-17 20:08:01
'''
from aliyun.api.base import RestApi
class Mts20140618SearchVideoListRequest(RestApi):
	def __init__(self,domain='mts.aliyuncs.com',port=80):
		RestApi.__init__(self,domain, port)
		self.OwnerId = None
		self.OwnerAccount = None
		self.ResourceOwnerAccount = None
		self.MediaState = None
		self.PageNumber = None
		self.PageSize = None

	def getapiname(self):
		return 'mts.aliyuncs.com.SearchVideoList.2014-06-18'
