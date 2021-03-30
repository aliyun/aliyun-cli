package cloudcallcenter

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

// SkillGroupData is a nested struct in cloudcallcenter response
type SkillGroupData struct {
	RecordDate                 string `json:"RecordDate" xml:"RecordDate"`
	LoggedInAgentNumber        string `json:"LoggedInAgentNumber" xml:"LoggedInAgentNumber"`
	AverageTalkTime            string `json:"AverageTalkTime" xml:"AverageTalkTime"`
	AverageTalkPercentage      string `json:"AverageTalkPercentage" xml:"AverageTalkPercentage"`
	NotReadyAgentNumber        string `json:"NotReadyAgentNumber" xml:"NotReadyAgentNumber"`
	InboundCallNumber          string `json:"InboundCallNumber" xml:"InboundCallNumber"`
	AverageAgentTalkTime       string `json:"AverageAgentTalkTime" xml:"AverageAgentTalkTime"`
	ReadyAgentNumber           string `json:"ReadyAgentNumber" xml:"ReadyAgentNumber"`
	OutboundCallNumber         string `json:"OutboundCallNumber" xml:"OutboundCallNumber"`
	OutboundAppraisePercentage string `json:"OutboundAppraisePercentage" xml:"OutboundAppraisePercentage"`
	TalkAgentNumber            string `json:"TalkAgentNumber" xml:"TalkAgentNumber"`
	MaxCallWaitTime            string `json:"MaxCallWaitTime" xml:"MaxCallWaitTime"`
	SkillGroupName             string `json:"SkillGroupName" xml:"SkillGroupName"`
	AppraisePercentage         string `json:"AppraisePercentage" xml:"AppraisePercentage"`
	SkillGroupId               string `json:"SkillGroupId" xml:"SkillGroupId"`
	InboundAppraisePercentage  string `json:"InboundAppraisePercentage" xml:"InboundAppraisePercentage"`
	AnsweredIntr20Percentage   string `json:"AnsweredIntr20Percentage" xml:"AnsweredIntr20Percentage"`
	InstanceId                 string `json:"InstanceId" xml:"InstanceId"`
	CallWaitNumber             string `json:"CallWaitNumber" xml:"CallWaitNumber"`
	AverageLoginTime           string `json:"AverageLoginTime" xml:"AverageLoginTime"`
	AgentNumber                string `json:"AgentNumber" xml:"AgentNumber"`
}
