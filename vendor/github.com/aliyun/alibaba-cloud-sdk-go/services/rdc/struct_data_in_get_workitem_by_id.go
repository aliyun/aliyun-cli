package rdc

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

// DataInGetWorkitemById is a nested struct in rdc response
type DataInGetWorkitemById struct {
	AkProjectId           int           `json:"AkProjectId" xml:"AkProjectId"`
	AssignedTo            string        `json:"AssignedTo" xml:"AssignedTo"`
	AssignedToId          int           `json:"AssignedToId" xml:"AssignedToId"`
	AssignedToIdList      string        `json:"AssignedToIdList" xml:"AssignedToIdList"`
	AssignedToIds         string        `json:"AssignedToIds" xml:"AssignedToIds"`
	AssignedToMaps        string        `json:"AssignedToMaps" xml:"AssignedToMaps"`
	AssignedToStaffId     string        `json:"AssignedToStaffId" xml:"AssignedToStaffId"`
	AttachmentIds         string        `json:"AttachmentIds" xml:"AttachmentIds"`
	AttachmentList        string        `json:"AttachmentList" xml:"AttachmentList"`
	Attachmented          bool          `json:"Attachmented" xml:"Attachmented"`
	BlackListNotice       string        `json:"BlackListNotice" xml:"BlackListNotice"`
	ChangeLogList         string        `json:"ChangeLogList" xml:"ChangeLogList"`
	CommentList           string        `json:"CommentList" xml:"CommentList"`
	CommitDate            int64         `json:"CommitDate" xml:"CommitDate"`
	CreatedAt             int64         `json:"CreatedAt" xml:"CreatedAt"`
	Description           string        `json:"Description" xml:"Description"`
	Guid                  string        `json:"Guid" xml:"Guid"`
	Id                    int           `json:"Id" xml:"Id"`
	IdPath                string        `json:"IdPath" xml:"IdPath"`
	IgnoreCheck           bool          `json:"IgnoreCheck" xml:"IgnoreCheck"`
	IgnoreIntegrate       bool          `json:"IgnoreIntegrate" xml:"IgnoreIntegrate"`
	IssueTypeId           int           `json:"IssueTypeId" xml:"IssueTypeId"`
	LogicalStatus         string        `json:"LogicalStatus" xml:"LogicalStatus"`
	ModuleIds             string        `json:"ModuleIds" xml:"ModuleIds"`
	ModuleList            string        `json:"ModuleList" xml:"ModuleList"`
	ModuleUpdated         bool          `json:"ModuleUpdated" xml:"ModuleUpdated"`
	ParentId              int           `json:"ParentId" xml:"ParentId"`
	Priority              string        `json:"Priority" xml:"Priority"`
	PriorityId            int           `json:"PriorityId" xml:"PriorityId"`
	ProjectIds            string        `json:"ProjectIds" xml:"ProjectIds"`
	RecordChangeLog       bool          `json:"RecordChangeLog" xml:"RecordChangeLog"`
	RegionId              int           `json:"RegionId" xml:"RegionId"`
	RelatedAKProjectGuids string        `json:"RelatedAKProjectGuids" xml:"RelatedAKProjectGuids"`
	RelatedAKProjectIds   string        `json:"RelatedAKProjectIds" xml:"RelatedAKProjectIds"`
	RelatedUserIds        string        `json:"RelatedUserIds" xml:"RelatedUserIds"`
	SendWangwang          bool          `json:"SendWangwang" xml:"SendWangwang"`
	SeriousLevel          string        `json:"SeriousLevel" xml:"SeriousLevel"`
	SeriousLevelId        int           `json:"SeriousLevelId" xml:"SeriousLevelId"`
	SkipCollab            bool          `json:"SkipCollab" xml:"SkipCollab"`
	Stamp                 string        `json:"Stamp" xml:"Stamp"`
	Status                string        `json:"Status" xml:"Status"`
	StatusId              int           `json:"StatusId" xml:"StatusId"`
	StatusStage           int           `json:"StatusStage" xml:"StatusStage"`
	Subject               string        `json:"Subject" xml:"Subject"`
	TagIdList             string        `json:"TagIdList" xml:"TagIdList"`
	TemplateId            int           `json:"TemplateId" xml:"TemplateId"`
	TrackerIds            string        `json:"TrackerIds" xml:"TrackerIds"`
	Trackers              string        `json:"Trackers" xml:"Trackers"`
	UpdateStatusAt        int64         `json:"UpdateStatusAt" xml:"UpdateStatusAt"`
	UpdatedAt             int64         `json:"UpdatedAt" xml:"UpdatedAt"`
	User                  string        `json:"User" xml:"User"`
	UserId                int           `json:"UserId" xml:"UserId"`
	UserStaffId           string        `json:"UserStaffId" xml:"UserStaffId"`
	Verifier              string        `json:"Verifier" xml:"Verifier"`
	VerifierId            int           `json:"VerifierId" xml:"VerifierId"`
	VerifierStaffId       string        `json:"VerifierStaffId" xml:"VerifierStaffId"`
	VersionIds            string        `json:"VersionIds" xml:"VersionIds"`
	VersionList           string        `json:"VersionList" xml:"VersionList"`
	Watched               bool          `json:"Watched" xml:"Watched"`
	CfsList               []CfsListItem `json:"CfsList" xml:"CfsList"`
}
