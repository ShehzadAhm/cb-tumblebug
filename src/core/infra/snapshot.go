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

// Package mci is to manage multi-cloud infra
package infra

import (
	"encoding/json"
	"fmt"

	"github.com/cloud-barista/cb-tumblebug/src/core/common"
	"github.com/cloud-barista/cb-tumblebug/src/core/model"
	"github.com/cloud-barista/cb-tumblebug/src/core/resource"
	"github.com/cloud-barista/cb-tumblebug/src/kvstore/kvstore"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// CreateVmSnapshot is func to create VM snapshot
func CreateVmSnapshot(nsId string, mciId string, vmId string, snapshotName string) (model.TbCustomImageInfo, error) {
	vmKey := common.GenMciKey(nsId, mciId, vmId)

	// Check existence of the key. If no key, no update.
	keyValue, err := kvstore.GetKv(vmKey)
	if keyValue == (kvstore.KeyValue{}) || err != nil {
		err := fmt.Errorf("Failed to find 'ns/mci/vm': %s/%s/%s \n", nsId, mciId, vmId)
		log.Error().Err(err).Msg("")
		return model.TbCustomImageInfo{}, err
	}

	vm := model.TbVmInfo{}
	json.Unmarshal([]byte(keyValue.Value), &vm)

	if snapshotName == "" {
		snapshotName = common.GenUid()
	}

	requestBody := model.SpiderMyImageReq{
		ConnectionName: vm.ConnectionName,
		ReqInfo: struct {
			Name     string
			SourceVM string
		}{
			Name:     snapshotName,
			SourceVM: vm.CspViewVmDetail.IId.NameId,
		},
	}

	client := resty.New().SetCloseConnection(true)
	client.SetAllowGetMethodPayload(true)

	req := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		SetResult(&model.SpiderMyImageInfo{}) // or SetResult(AuthSuccess{}).
		//SetError(&AuthError{}).       // or SetError(AuthError{}).

	// Inspect DataDisks before creating VM snapshot
	// Disabled because: there is no difference in dataDisks before and after creating VM snapshot
	// inspect_result_before_snapshot, err := InspectResources(vm.ConnectionName, model.StrDataDisk)
	// dataDisks_before_snapshot := inspect_result_before_snapshot.Resources.OnTumblebug.Info
	// if err != nil {
	// 	err := fmt.Errorf("Failed to get current datadisks' info. \n")
	// 	log.Error().Err(err).Msg("")
	// 	return model.TbCustomImageInfo{}, err
	// }

	// Create VM snapshot
	url := fmt.Sprintf("%s/myimage", model.SpiderRestUrl)

	resp, err := req.Post(url)

	fmt.Printf("HTTP Status code: %d \n", resp.StatusCode())
	switch {
	case resp.StatusCode() >= 400 || resp.StatusCode() < 200:
		err := fmt.Errorf(string(resp.Body()))
		fmt.Println("body: ", string(resp.Body()))
		log.Error().Err(err).Msg("")
		return model.TbCustomImageInfo{}, err
	}

	// create one customImage
	tempSpiderMyImageInfo := resp.Result().(*model.SpiderMyImageInfo)
	tempTbCustomImageInfo := model.TbCustomImageInfo{
		Namespace:            nsId,
		Id:                   "", // This field will be assigned in RegisterCustomImageWithInfo()
		Name:                 snapshotName,
		ConnectionName:       vm.ConnectionName,
		SourceVmId:           vmId,
		CspCustomImageId:     tempSpiderMyImageInfo.IId.SystemId,
		CspCustomImageName:   tempSpiderMyImageInfo.IId.NameId,
		Description:          "",
		CreationDate:         tempSpiderMyImageInfo.CreatedTime,
		GuestOS:              "",
		Status:               tempSpiderMyImageInfo.Status,
		KeyValueList:         tempSpiderMyImageInfo.KeyValueList,
		AssociatedObjectList: []string{},
		IsAutoGenerated:      false,
		SystemLabel:          "",
	}

	result, err := resource.RegisterCustomImageWithInfo(nsId, tempTbCustomImageInfo)
	if err != nil {
		err := fmt.Errorf("Failed to find 'ns/mci/vm': %s/%s/%s \n", nsId, mciId, vmId)
		log.Error().Err(err).Msg("")
		return model.TbCustomImageInfo{}, err
	}

	// Inspect DataDisks after creating VM snapshot
	// Disabled because: there is no difference in dataDisks before and after creating VM snapshot
	// inspect_result_after_snapshot, err := InspectResources(vm.ConnectionName, model.StrDataDisk)
	// dataDisks_after_snapshot := inspect_result_after_snapshot.Resources.OnTumblebug.Info
	// if err != nil {
	// 	err := fmt.Errorf("Failed to get current datadisks' info. \n")
	// 	log.Error().Err(err).Msg("")
	// 	return model.TbCustomImageInfo{}, err
	// }

	// difference_dataDisks := Difference_dataDisks(dataDisks_before_snapshot, dataDisks_after_snapshot)

	// // create 'n' dataDisks
	// for _, v := range difference_dataDisks {
	// 	tempTbDataDiskReq := model.TbDataDiskReq{
	// 		Name:           fmt.Sprintf("%s-%s", vm.Name, common.GenerateNewRandomString(5)),
	// 		ConnectionName: vm.ConnectionName,
	// 		CspDataDiskId:  v.IdByCsp,
	// 	}

	// 	_, err = resource.CreateDataDisk(nsId, &tempTbDataDiskReq, "register")
	// 	if err != nil {
	// 		err := fmt.Errorf("Failed to register the created dataDisk %s to TB. \n", v.IdByCsp)
	// 		log.Error().Err(err).Msg("")
	// 		continue
	// 	}
	// }

	return result, nil
}

func Difference_dataDisks(a, b []model.ResourceOnTumblebugInfo) []model.ResourceOnTumblebugInfo {
	mb := make(map[interface{}]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []model.ResourceOnTumblebugInfo
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
