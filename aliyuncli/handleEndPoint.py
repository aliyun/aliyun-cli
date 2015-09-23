__author__ = 'zhaoyang.szy'

def handleEndPoint(cmd,operation,keyValues):
    if not hasNecessaryArgs(keyValues):
        print 'RegionId/EndPoint is absence'
        return
    if cmd is not None:
        cmd = cmd.capitalize()
    regionId = getRegionId(keyValues)
    endPoint = getEndPoint(keyValues)
    _handleEndPoint(cmd,regionId,endPoint)


def hasNecessaryArgs(keyValues):
    if keyValues is not None and isinstance(keyValues,dict):
        if keyValues.get('RegionId') is not None and keyValues.get('EndPoint') is not None:
            return True
    else:
        return False



def _handleEndPoint(cmd,regionId,endPoint):
    try:
        from aliyunsdkcore.profile.region_provider import modify_point
        modify_point(cmd,regionId,endPoint)
    except Exception as e:
        print e
        pass


def getRegionId(keyValues):
    if keyValues is not None and isinstance(keyValues,dict):
        regionId = keyValues.get('RegionId')
        if regionId is not None and isinstance(regionId,list) and len(regionId)>0:
            return regionId[0]

def getEndPoint(keyValues):
    if keyValues is not None and isinstance(keyValues,dict):
        endPoint = keyValues.get('EndPoint')
        if endPoint is not None and isinstance(endPoint,list) and len(endPoint)>0:
            return  endPoint[0]
