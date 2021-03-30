package r_kvstore

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

// DBInstanceAttribute is a nested struct in r_kvstore response
type DBInstanceAttribute struct {
	InstanceId                string                          `json:"InstanceId" xml:"InstanceId"`
	InstanceName              string                          `json:"InstanceName" xml:"InstanceName"`
	ConnectionDomain          string                          `json:"ConnectionDomain" xml:"ConnectionDomain"`
	Port                      int64                           `json:"Port" xml:"Port"`
	InstanceStatus            string                          `json:"InstanceStatus" xml:"InstanceStatus"`
	RegionId                  string                          `json:"RegionId" xml:"RegionId"`
	Capacity                  int64                           `json:"Capacity" xml:"Capacity"`
	InstanceClass             string                          `json:"InstanceClass" xml:"InstanceClass"`
	QPS                       int64                           `json:"QPS" xml:"QPS"`
	Bandwidth                 int64                           `json:"Bandwidth" xml:"Bandwidth"`
	Connections               int64                           `json:"Connections" xml:"Connections"`
	ZoneId                    string                          `json:"ZoneId" xml:"ZoneId"`
	Config                    string                          `json:"Config" xml:"Config"`
	ChargeType                string                          `json:"ChargeType" xml:"ChargeType"`
	NodeType                  string                          `json:"NodeType" xml:"NodeType"`
	NetworkType               string                          `json:"NetworkType" xml:"NetworkType"`
	VpcId                     string                          `json:"VpcId" xml:"VpcId"`
	VSwitchId                 string                          `json:"VSwitchId" xml:"VSwitchId"`
	PrivateIp                 string                          `json:"PrivateIp" xml:"PrivateIp"`
	CreateTime                string                          `json:"CreateTime" xml:"CreateTime"`
	EndTime                   string                          `json:"EndTime" xml:"EndTime"`
	HasRenewChangeOrder       string                          `json:"HasRenewChangeOrder" xml:"HasRenewChangeOrder"`
	IsRds                     bool                            `json:"IsRds" xml:"IsRds"`
	Engine                    string                          `json:"Engine" xml:"Engine"`
	EngineVersion             string                          `json:"EngineVersion" xml:"EngineVersion"`
	MaintainStartTime         string                          `json:"MaintainStartTime" xml:"MaintainStartTime"`
	MaintainEndTime           string                          `json:"MaintainEndTime" xml:"MaintainEndTime"`
	AvailabilityValue         string                          `json:"AvailabilityValue" xml:"AvailabilityValue"`
	SecurityIPList            string                          `json:"SecurityIPList" xml:"SecurityIPList"`
	InstanceType              string                          `json:"InstanceType" xml:"InstanceType"`
	ArchitectureType          string                          `json:"ArchitectureType" xml:"ArchitectureType"`
	PackageType               string                          `json:"PackageType" xml:"PackageType"`
	ReplicaId                 string                          `json:"ReplicaId" xml:"ReplicaId"`
	VpcAuthMode               string                          `json:"VpcAuthMode" xml:"VpcAuthMode"`
	AuditLogRetention         string                          `json:"AuditLogRetention" xml:"AuditLogRetention"`
	ReplicationMode           string                          `json:"ReplicationMode" xml:"ReplicationMode"`
	VpcCloudInstanceId        string                          `json:"VpcCloudInstanceId" xml:"VpcCloudInstanceId"`
	InstanceReleaseProtection bool                            `json:"InstanceReleaseProtection" xml:"InstanceReleaseProtection"`
	ResourceGroupId           string                          `json:"ResourceGroupId" xml:"ResourceGroupId"`
	ShardCount                int                             `json:"ShardCount" xml:"ShardCount"`
	Storage                   string                          `json:"Storage" xml:"Storage"`
	StorageType               string                          `json:"StorageType" xml:"StorageType"`
	GlobalInstanceId          string                          `json:"GlobalInstanceId" xml:"GlobalInstanceId"`
	Tags                      TagsInDescribeInstanceAttribute `json:"Tags" xml:"Tags"`
}
