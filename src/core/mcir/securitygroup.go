/*
Copyright 2019 The Cloud-Barista Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package mcir is to manage multi-cloud infra resource
package mcir

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cloud-barista/cb-spider/interface/api"
	"github.com/cloud-barista/cb-tumblebug/src/core/common"
	validator "github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
)

// SpiderSecurityReqInfoWrapper is a wrapper struct to create JSON body of 'Create security group request'
type SpiderSecurityReqInfoWrapper struct {
	ConnectionName string
	ReqInfo        SpiderSecurityInfo
}

// SpiderSecurityRuleReqInfoWrapper is a wrapper struct to create JSON body of 'Create security rule'
type SpiderSecurityRuleReqInfoWrapper struct {
	ConnectionName string
	ReqInfo        SpiderSecurityRuleReqInfoIntermediateWrapper
}

// SpiderSecurityRuleReqInfoIntermediateWrapper is a intermediate wrapper struct between SpiderSecurityRuleReqInfoWrapper and SpiderSecurityRuleInfo.
type SpiderSecurityRuleReqInfoIntermediateWrapper struct {
	RuleInfoList []SpiderSecurityRuleInfo
}

// SpiderSecurityRuleInfo is a struct to handle security group rule info from/to CB-Spider.
type SpiderSecurityRuleInfo struct {
	FromPort   string //`json:"fromPort"`
	ToPort     string //`json:"toPort"`
	IPProtocol string //`json:"ipProtocol"`
	Direction  string //`json:"direction"`
	CIDR       string
}

// SpiderSecurityRuleInfo is a struct to create JSON body of 'Create security group request'
type SpiderSecurityInfo struct {
	// Fields for request
	Name    string
	VPCName string
	CSPId   string

	// Fields for both request and response
	SecurityRules []SpiderSecurityRuleInfo

	// Fields for response
	IId          common.IID // {NameId, SystemId}
	VpcIID       common.IID // {NameId, SystemId}
	Direction    string     // @todo userd??
	KeyValueList []common.KeyValue
}

// TbSecurityGroupReq is a struct to handle 'Create security group' request toward CB-Tumblebug.
type TbSecurityGroupReq struct { // Tumblebug
	Name           string                `json:"name" validate:"required"`
	ConnectionName string                `json:"connectionName" validate:"required"`
	VNetId         string                `json:"vNetId" validate:"required"`
	Description    string                `json:"description"`
	FirewallRules  *[]TbFirewallRuleInfo `json:"firewallRules"` // validate:"required"`

	// CspSecurityGroupId is required to register object from CSP (option=register)
	CspSecurityGroupId string `json:"cspSecurityGroupId"`
}

// TbFirewallRuleInfo is a struct to handle firewall rule info of CB-Tumblebug.
type TbFirewallRuleInfo struct {
	FromPort   string `validate:"required"` //`json:"fromPort"`
	ToPort     string `validate:"required"` //`json:"toPort"`
	IPProtocol string `validate:"required"` //`json:"ipProtocol"`
	Direction  string `validate:"required"` //`json:"direction"`
	CIDR       string
}

// TbSecurityGroupReqStructLevelValidation is a function to validate 'TbSecurityGroupReq' object.
func TbSecurityGroupReqStructLevelValidation(sl validator.StructLevel) {

	u := sl.Current().Interface().(TbSecurityGroupReq)

	err := common.CheckString(u.Name)
	if err != nil {
		// ReportError(field interface{}, fieldName, structFieldName, tag, param string)
		sl.ReportError(u.Name, "name", "Name", err.Error(), "")
	}
}

// TbSecurityGroupInfo is a struct that represents TB security group object.
type TbSecurityGroupInfo struct { // Tumblebug
	Id                   string               `json:"id"`
	Name                 string               `json:"name"`
	ConnectionName       string               `json:"connectionName"`
	VNetId               string               `json:"vNetId"`
	Description          string               `json:"description"`
	FirewallRules        []TbFirewallRuleInfo `json:"firewallRules"`
	CspSecurityGroupId   string               `json:"cspSecurityGroupId"`
	CspSecurityGroupName string               `json:"cspSecurityGroupName"`
	KeyValueList         []common.KeyValue    `json:"keyValueList"`
	AssociatedObjectList []string             `json:"associatedObjectList"`
	IsAutoGenerated      bool                 `json:"isAutoGenerated"`

	// SystemLabel is for describing the MCIR in a keyword (any string can be used) for special System purpose
	SystemLabel string `json:"systemLabel" example:"Managed by CB-Tumblebug" default:""`

	// Disabled for now
	//ResourceGroupName  string `json:"resourceGroupName"`
}

// CreateSecurityGroup accepts SG creation request, creates and returns an TB SG object
func CreateSecurityGroup(nsId string, u *TbSecurityGroupReq, option string) (TbSecurityGroupInfo, error) {

	resourceType := common.StrSecurityGroup

	err := common.CheckString(nsId)
	if err != nil {
		temp := TbSecurityGroupInfo{}
		common.CBLog.Error(err)
		return temp, err
	}

	// if option == "register" {
	// 	mockFirewallRule := SpiderSecurityRuleInfo{
	// 		FromPort:   "22",
	// 		ToPort:     "22",
	// 		IPProtocol: "tcp",
	// 		Direction:  "inbound",
	// 		CIDR:       "0.0.0.0/0",
	// 	}

	// 	*u.FirewallRules = append(*u.FirewallRules, mockFirewallRule)
	// }

	if option != "register" {
		err = validate.Var(u.FirewallRules, "required")
		if err != nil {
			temp := TbSecurityGroupInfo{}
			if _, ok := err.(*validator.InvalidValidationError); ok {
				fmt.Println(err)
				return temp, err
			}
			return temp, err
		}
	}

	err = validate.Struct(u)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			fmt.Println(err)
			temp := TbSecurityGroupInfo{}
			return temp, err
		}

		temp := TbSecurityGroupInfo{}
		return temp, err
	}

	check, err := CheckResource(nsId, resourceType, u.Name)

	if check {
		temp := TbSecurityGroupInfo{}
		err := fmt.Errorf("The securityGroup " + u.Name + " already exists.")
		return temp, err
	}
	if err != nil {
		common.CBLog.Error(err)
		content := TbSecurityGroupInfo{}
		err := fmt.Errorf("Cannot create securityGroup")
		return content, err
	}

	// TODO: Need to be improved
	// Avoid retrieving vNet info if option == register
	// Assign random temporal ID to u.VNetId
	if option == "register" && u.VNetId == "not defined" {
		resourceList, err := ListResource(nsId, common.StrVNet)

		if err != nil {
			common.CBLog.Error(err)
			err := fmt.Errorf("Cannot list vNet Ids for securityGroup")
			return TbSecurityGroupInfo{}, err
		}

		var content struct {
			VNet []TbVNetInfo `json:"vNet"`
		}
		content.VNet = resourceList.([]TbVNetInfo) // type assertion (interface{} -> array)

		if len(content.VNet) == 0 {
			errString := "There is no " + common.StrVNet + " resource in " + nsId
			err := fmt.Errorf(errString)
			common.CBLog.Error(err)
			return TbSecurityGroupInfo{}, err
		}

		// Assign random temporal ID to u.VNetId (should be in the same Connection with SG)
		for _, r := range content.VNet {
			if r.ConnectionName == u.ConnectionName {
				u.VNetId = r.Id
			}
		}
	}

	vNetInfo := TbVNetInfo{}
	tempInterface, err := GetResource(nsId, common.StrVNet, u.VNetId)
	if err != nil {
		err := fmt.Errorf("Failed to get the TbVNetInfo " + u.VNetId + ".")
		return TbSecurityGroupInfo{}, err
	}
	err = common.CopySrcToDest(&tempInterface, &vNetInfo)
	if err != nil {
		err := fmt.Errorf("Failed to get the TbVNetInfo-CopySrcToDest() " + u.VNetId + ".")
		return TbSecurityGroupInfo{}, err
	}

	tempReq := SpiderSecurityReqInfoWrapper{}
	tempReq.ConnectionName = u.ConnectionName
	tempReq.ReqInfo.Name = fmt.Sprintf("%s-%s", nsId, u.Name)
	tempReq.ReqInfo.VPCName = vNetInfo.CspVNetName
	tempReq.ReqInfo.CSPId = u.CspSecurityGroupId

	// tempReq.ReqInfo.SecurityRules = u.FirewallRules
	if u.FirewallRules != nil {
		for _, v := range *u.FirewallRules {
			jsonBody, err := json.Marshal(v)
			if err != nil {
				common.CBLog.Error(err)
			}

			spiderSecurityRuleInfo := SpiderSecurityRuleInfo{}
			err = json.Unmarshal(jsonBody, &spiderSecurityRuleInfo)
			if err != nil {
				common.CBLog.Error(err)
			}

			tempReq.ReqInfo.SecurityRules = append(tempReq.ReqInfo.SecurityRules, spiderSecurityRuleInfo)
		}
	}

	var tempSpiderSecurityInfo *SpiderSecurityInfo

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		client := resty.New().SetCloseConnection(true)
		client.SetAllowGetMethodPayload(true)

		req := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderSecurityInfo{}) // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).

		var resp *resty.Response
		var err error

		var url string
		if option == "register" && u.CspSecurityGroupId == "" {
			url = fmt.Sprintf("%s/securitygroup/%s", common.SpiderRestUrl, u.Name)
			resp, err = req.Get(url)
		} else if option == "register" && u.CspSecurityGroupId != "" {
			url = fmt.Sprintf("%s/regsecuritygroup", common.SpiderRestUrl)
			resp, err = req.Post(url)
		} else { // option != "register"
			url = fmt.Sprintf("%s/securitygroup", common.SpiderRestUrl)
			resp, err = req.Post(url)
		}

		if err != nil {
			common.CBLog.Error(err)
			content := TbSecurityGroupInfo{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println("HTTP Status code: " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			common.CBLog.Error(err)
			content := TbSecurityGroupInfo{}
			return content, err
		}

		tempSpiderSecurityInfo = resp.Result().(*SpiderSecurityInfo)

	} else {

		// Set CCM gRPC API
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return TbSecurityGroupInfo{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return TbSecurityGroupInfo{}, err
		}
		defer ccm.Close()

		payload, _ := json.Marshal(tempReq)

		//result, err := ccm.CreateSecurity(string(payload))
		var result string

		if option == "register" {
			result, err = ccm.CreateVPC(string(payload))
		} else {
			result, err = ccm.GetVPC(string(payload))
		}

		if err != nil {
			common.CBLog.Error(err)
			return TbSecurityGroupInfo{}, err
		}

		tempSpiderSecurityInfo = &SpiderSecurityInfo{}
		err = json.Unmarshal([]byte(result), &tempSpiderSecurityInfo)
		if err != nil {
			common.CBLog.Error(err)
			return TbSecurityGroupInfo{}, err
		}
	}

	content := TbSecurityGroupInfo{}
	content.Id = u.Name
	content.Name = u.Name
	content.ConnectionName = u.ConnectionName
	content.VNetId = tempSpiderSecurityInfo.VpcIID.NameId
	content.CspSecurityGroupId = tempSpiderSecurityInfo.IId.SystemId
	content.CspSecurityGroupName = tempSpiderSecurityInfo.IId.NameId
	content.Description = u.Description
	content.KeyValueList = tempSpiderSecurityInfo.KeyValueList
	content.AssociatedObjectList = []string{}

	// content.FirewallRules = tempSpiderSecurityInfo.SecurityRules
	tempTbFirewallRules := []TbFirewallRuleInfo{}
	for _, v := range tempSpiderSecurityInfo.SecurityRules {
		tempTbFirewallRule := TbFirewallRuleInfo(v) // type casting
		tempTbFirewallRules = append(tempTbFirewallRules, tempTbFirewallRule)
	}
	content.FirewallRules = tempTbFirewallRules

	if option == "register" && u.CspSecurityGroupId == "" {
		content.SystemLabel = "Registered from CB-Spider resource"
	} else if option == "register" && u.CspSecurityGroupId != "" {
		content.SystemLabel = "Registered from CSP resource"
	}

	// cb-store
	fmt.Println("=========================== PUT CreateSecurityGroup")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(Key, string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return content, err
	}

	keyValue, err := common.CBStore.Get(Key)
	if err != nil {
		common.CBLog.Error(err)
		err = fmt.Errorf("In CreateSecurityGroup(); CBStore.Get() returned an error.")
		common.CBLog.Error(err)
		// return nil, err
	}

	fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")
	return content, nil
}
