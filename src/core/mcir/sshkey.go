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

type SpiderKeyPairReqInfoWrapper struct { // Spider
	ConnectionName string
	ReqInfo        SpiderKeyPairInfo
}

type SpiderKeyPairInfo struct { // Spider
	// Fields for request
	Name string

	// Fields for response
	IId          common.IID // {NameId, SystemId}
	Fingerprint  string
	PublicKey    string
	PrivateKey   string
	VMUserID     string
	KeyValueList []common.KeyValue
}

type TbSshKeyReq struct {
	Name           string `json:"name" validate:"required"`
	ConnectionName string `json:"connectionName" validate:"required"`
	Description    string `json:"description"`
}

func TbSshKeyReqStructLevelValidation(sl validator.StructLevel) {

	u := sl.Current().Interface().(TbSshKeyReq)

	err := common.CheckString(u.Name)
	if err != nil {
		// ReportError(field interface{}, fieldName, structFieldName, tag, param string)
		sl.ReportError(u.Name, "name", "Name", "NotObeyingNamingConvention", "")
	}
}

type TbSshKeyInfo struct {
	Id                   string            `json:"id"`
	Name                 string            `json:"name"`
	ConnectionName       string            `json:"connectionName"`
	Description          string            `json:"description"`
	CspSshKeyName        string            `json:"cspSshKeyName"`
	Fingerprint          string            `json:"fingerprint"`
	Username             string            `json:"username"`
	VerifiedUsername     string            `json:"verifiedUsername"`
	PublicKey            string            `json:"publicKey"`
	PrivateKey           string            `json:"privateKey"`
	KeyValueList         []common.KeyValue `json:"keyValueList"`
	AssociatedObjectList []string          `json:"associatedObjectList"`
	IsAutoGenerated      bool              `json:"isAutoGenerated"`
}

// CreateSshKey accepts SSH key creation request, creates and returns an TB sshKey object
func CreateSshKey(nsId string, u *TbSshKeyReq) (TbSshKeyInfo, error) {

	resourceType := common.StrSSHKey

	err := common.CheckString(nsId)
	if err != nil {
		temp := TbSshKeyInfo{}
		common.CBLog.Error(err)
		return temp, err
	}

	// returns InvalidValidationError for bad validation input, nil or ValidationErrors ( []FieldError )
	err = validate.Struct(u)
	if err != nil {

		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		if _, ok := err.(*validator.InvalidValidationError); ok {
			fmt.Println(err)
			temp := TbSshKeyInfo{}
			return temp, err
		}

		// for _, err := range err.(validator.ValidationErrors) {

		// 	fmt.Println(err.Namespace()) // can differ when a custom TagNameFunc is registered or
		// 	fmt.Println(err.Field())     // by passing alt name to ReportError like below
		// 	fmt.Println(err.StructNamespace())
		// 	fmt.Println(err.StructField())
		// 	fmt.Println(err.Tag())
		// 	fmt.Println(err.ActualTag())
		// 	fmt.Println(err.Kind())
		// 	fmt.Println(err.Type())
		// 	fmt.Println(err.Value())
		// 	fmt.Println(err.Param())
		// 	fmt.Println()
		// }

		temp := TbSshKeyInfo{}
		return temp, err
	}

	check, err := CheckResource(nsId, resourceType, u.Name)

	if check {
		temp := TbSshKeyInfo{}
		err := fmt.Errorf("The sshKey " + u.Name + " already exists.")
		//return temp, http.StatusConflict, nil, err
		return temp, err
	}

	if err != nil {
		temp := TbSshKeyInfo{}
		err := fmt.Errorf("Failed to check the existence of the sshKey " + u.Name + ".")
		return temp, err
	}

	tempReq := SpiderKeyPairReqInfoWrapper{}
	tempReq.ConnectionName = u.ConnectionName
	tempReq.ReqInfo.Name = u.Name

	var tempSpiderKeyPairInfo *SpiderKeyPairInfo

	if os.Getenv("SPIDER_CALL_METHOD") == "REST" {

		url := common.SpiderRestUrl + "/keypair"

		client := resty.New().SetCloseConnection(true)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetBody(tempReq).
			SetResult(&SpiderKeyPairInfo{}). // or SetResult(AuthSuccess{}).
			//SetError(&AuthError{}).       // or SetError(AuthError{}).
			Post(url)

		if err != nil {
			common.CBLog.Error(err)
			content := TbSshKeyInfo{}
			err := fmt.Errorf("an error occurred while requesting to CB-Spider")
			return content, err
		}

		fmt.Println("HTTP Status code: " + strconv.Itoa(resp.StatusCode()))
		switch {
		case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
			err := fmt.Errorf(string(resp.Body()))
			fmt.Println("body: ", string(resp.Body()))
			common.CBLog.Error(err)
			content := TbSshKeyInfo{}
			return content, err
		}

		tempSpiderKeyPairInfo = resp.Result().(*SpiderKeyPairInfo)

	} else {

		// Set CCM gRPC API
		ccm := api.NewCloudResourceHandler()
		err := ccm.SetConfigPath(os.Getenv("CBTUMBLEBUG_ROOT") + "/conf/grpc_conf.yaml")
		if err != nil {
			common.CBLog.Error("ccm failed to set config : ", err)
			return TbSshKeyInfo{}, err
		}
		err = ccm.Open()
		if err != nil {
			common.CBLog.Error("ccm api open failed : ", err)
			return TbSshKeyInfo{}, err
		}
		defer ccm.Close()

		payload, _ := json.MarshalIndent(tempReq, "", "  ")
		//fmt.Println("payload: " + string(payload)) // for debug

		result, err := ccm.CreateKey(string(payload))
		if err != nil {
			common.CBLog.Error(err)
			return TbSshKeyInfo{}, err
		}

		tempSpiderKeyPairInfo = &SpiderKeyPairInfo{}
		err = json.Unmarshal([]byte(result), &tempSpiderKeyPairInfo)
		if err != nil {
			common.CBLog.Error(err)
			return TbSshKeyInfo{}, err
		}

	}

	content := TbSshKeyInfo{}
	//content.Id = common.GenUuid()
	content.Id = u.Name
	content.Name = u.Name
	content.ConnectionName = u.ConnectionName
	content.CspSshKeyName = tempSpiderKeyPairInfo.IId.NameId
	content.Fingerprint = tempSpiderKeyPairInfo.Fingerprint
	content.Username = tempSpiderKeyPairInfo.VMUserID
	content.PublicKey = tempSpiderKeyPairInfo.PublicKey
	content.PrivateKey = tempSpiderKeyPairInfo.PrivateKey
	content.Description = u.Description
	content.KeyValueList = tempSpiderKeyPairInfo.KeyValueList
	content.AssociatedObjectList = []string{}

	// cb-store
	fmt.Println("=========================== PUT CreateSshKey")
	Key := common.GenResourceKey(nsId, resourceType, content.Id)
	Val, _ := json.Marshal(content)
	err = common.CBStore.Put(string(Key), string(Val))
	if err != nil {
		common.CBLog.Error(err)
		return content, err
	}
	//keyValue, _ := common.CBStore.Get(string(Key))
	//fmt.Println("<" + keyValue.Key + "> \n" + keyValue.Value)
	fmt.Println("===========================")
	return content, nil
}
