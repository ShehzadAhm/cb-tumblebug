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

// Package model is to handle object of CB-Tumblebug
package model

// SpiderKeyPairReqInfoWrapper is a wrapper struct to create JSON body of 'Create keypair request'
type SpiderKeyPairReqInfoWrapper struct {
	ConnectionName string
	ReqInfo        SpiderKeyPairInfo
}

// SpiderKeyPairInfo is a struct to create JSON body of 'Create keypair request'
type SpiderKeyPairInfo struct {
	// Fields for request
	Name  string
	CSPId string

	// Fields for response
	IId          IID // {NameId, SystemId}
	Fingerprint  string
	PublicKey    string
	PrivateKey   string
	VMUserID     string
	KeyValueList []KeyValue
}

// TbSshKeyReq is a struct to handle 'Create SSH key' request toward CB-Tumblebug.
type TbSshKeyReq struct {
	Name           string `json:"name" validate:"required"`
	ConnectionName string `json:"connectionName" validate:"required"`
	Description    string `json:"description"`

	// Fields for "Register existing SSH keys" feature
	// CspSshKeyId is required to register object from CSP (option=register)
	CspSshKeyId      string `json:"cspSshKeyId"`
	Fingerprint      string `json:"fingerprint"`
	Username         string `json:"username"`
	VerifiedUsername string `json:"verifiedUsername"`
	PublicKey        string `json:"publicKey"`
	PrivateKey       string `json:"privateKey"`
}

// TbSshKeyInfo is a struct that represents TB SSH key object.
type TbSshKeyInfo struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// uuid is universally unique identifier for the resource
	Uuid           string `json:"uuid,omitempty"`
	ConnectionName string `json:"connectionName,omitempty"`
	Description    string `json:"description,omitempty"`

	// CspSshKeyId used for CSP-native identifier (either Name or ID)
	CspSshKeyId string `json:"cspSshKeyId,omitempty"`

	// CspSshKeyName used for CB-Spider identifier
	CspSshKeyName string `json:"cspSshKeyName,omitempty"`

	Fingerprint          string     `json:"fingerprint,omitempty"`
	Username             string     `json:"username,omitempty"`
	VerifiedUsername     string     `json:"verifiedUsername,omitempty"`
	PublicKey            string     `json:"publicKey,omitempty"`
	PrivateKey           string     `json:"privateKey,omitempty"`
	KeyValueList         []KeyValue `json:"keyValueList,omitempty"`
	AssociatedObjectList []string   `json:"associatedObjectList,omitempty"`
	IsAutoGenerated      bool       `json:"isAutoGenerated,omitempty"`

	// SystemLabel is for describing the Resource in a keyword (any string can be used) for special System purpose
	SystemLabel string `json:"systemLabel,omitempty" example:"Managed by CB-Tumblebug" default:""`
}
