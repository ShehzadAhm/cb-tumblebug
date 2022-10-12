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

// Package mcis is to handle REST API for mcis
package mcis

import (
	"net/http"
	"strings"

	"github.com/cloud-barista/cb-tumblebug/src/core/mcir"
	"github.com/cloud-barista/cb-tumblebug/src/core/mcis"
	"github.com/labstack/echo/v4"
)

// RestGetControlMcis godoc
// @Summary Control the lifecycle of MCIS (refine, suspend, resume, reboot, terminate)
// @Description Control the lifecycle of MCIS (refine, suspend, resume, reboot, terminate)
// @Tags [Infra service] MCIS Control lifecycle
// @Accept  json
// @Produce  json
// @Param nsId path string true "Namespace ID" default(ns01)
// @Param mcisId path string true "MCIS ID" default(mcis01)
// @Param action query string true "Action to MCIS" Enums(suspend, resume, reboot, terminate, refine)
// @Param force query string false "Force control to skip checking controllable status" Enums(false, true)
// @Success 200 {object} common.SimpleMsg
// @Failure 404 {object} common.SimpleMsg
// @Failure 500 {object} common.SimpleMsg
// @Router /ns/{nsId}/control/mcis/{mcisId} [get]
func RestGetControlMcis(c echo.Context) error {
	nsId := c.Param("nsId")
	mcisId := c.Param("mcisId")

	action := c.QueryParam("action")
	force := c.QueryParam("force")
	forceOption := false
	if force == "true" {
		forceOption = true
	}

	if action == "suspend" || action == "resume" || action == "reboot" || action == "terminate" || action == "refine" {

		result, err := mcis.HandleMcisAction(nsId, mcisId, action, forceOption)
		if err != nil {
			mapA := map[string]string{"message": err.Error()}
			return c.JSON(http.StatusInternalServerError, &mapA)
		}

		mapA := map[string]string{"message": result}
		return c.JSON(http.StatusOK, &mapA)

	} else {
		mapA := map[string]string{"message": "'action' should be one of these: suspend, resume, reboot, terminate, refine"}
		return c.JSON(http.StatusBadRequest, &mapA)
	}
}

// RestGetControlMcisVm godoc
// @Summary Control the lifecycle of VM (suspend, resume, reboot, terminate)
// @Description Control the lifecycle of VM (suspend, resume, reboot, terminate)
// @Tags [Infra service] MCIS Control lifecycle
// @Accept  json
// @Produce  json
// @Param nsId path string true "Namespace ID" default(ns01)
// @Param mcisId path string true "MCIS ID" default(mcis01)
// @Param vmId path string true "VM ID" default(vm01)
// @Param action query string true "Action to MCIS" Enums(suspend, resume, reboot, terminate)
// @Success 200 {object} common.SimpleMsg
// @Failure 404 {object} common.SimpleMsg
// @Failure 500 {object} common.SimpleMsg
// @Router /ns/{nsId}/control/mcis/{mcisId}/vm/{vmId} [get]
func RestGetControlMcisVm(c echo.Context) error {

	nsId := c.Param("nsId")
	mcisId := c.Param("mcisId")
	vmId := c.Param("vmId")

	action := c.QueryParam("action")

	if action == "suspend" || action == "resume" || action == "reboot" || action == "terminate" {

		result, err := mcis.CoreGetMcisVmAction(nsId, mcisId, vmId, action)
		if err != nil {
			mapA := map[string]string{"message": err.Error()}
			return c.JSON(http.StatusInternalServerError, &mapA)
		}

		mapA := map[string]string{"message": result}
		return c.JSON(http.StatusOK, &mapA)

	} else {
		mapA := map[string]string{"message": "'action' should be one of these: suspend, resume, reboot, terminate"}
		return c.JSON(http.StatusBadRequest, &mapA)
	}
}

// RestPutMcisVmWithCmd godoc
// @Summary Attach/Detach data disk to/from VM
// @Description Attach/Detach data disk to/from VM
// @Tags [Infra resource] MCIR Data Disk management
// @Accept  json
// @Produce  json
// @Param nsId path string true "Namespace ID" default(ns01)
// @Param mcisId path string true "MCIS ID" default(mcis01)
// @Param vmId path string true "VM ID" default(vm01)
// @Param command path string true "Command to perform" Enums(attachDataDisk, detachDataDisk)
// @Param dataDisk body mcir.TbAttachDetachDataDiskReq true "Data disk ID to attach/detach"
// @Success 200 {object} mcis.TbVmInfo
// @Failure 404 {object} common.SimpleMsg
// @Failure 500 {object} common.SimpleMsg
// @Router /ns/{nsId}/mcis/{mcisId}/vm/{vmId}/{command} [put]
func RestPutMcisVmWithCmd(c echo.Context) error {

	nsId := c.Param("nsId")
	mcisId := c.Param("mcisId")
	vmId := c.Param("vmId")

	command := strings.Split(c.Path(), "/")[8]
	// c.Path(): /tumblebug/ns/:nsId/mcis/{mcisId}/vm/{vmId}/attachDataDisk

	u := &mcir.TbAttachDetachDataDiskReq{}
	if err := c.Bind(u); err != nil {
		return err
	}

	switch command {
	case mcis.AttachDataDisk:
		fallthrough
	case mcis.DetachDataDisk:
		result, err := mcis.AttachDetachDataDisk(nsId, mcisId, vmId, command, u.DataDiskId)
		if err != nil {
			mapA := map[string]string{"message": err.Error()}
			return c.JSON(http.StatusNotFound, &mapA)
		}

		// common.PrintJsonPretty(result)

		return c.JSON(http.StatusOK, result)
	default:
		mapA := map[string]string{"message": "Supported commands: attachDataDisk, detachDataDisk"}
		return c.JSON(http.StatusNotFound, &mapA)
	}
	return nil
}

// RestPostMcisVmWithCmd godoc
// @Summary Create VM snapshot
// @Description Create VM snapshot
// @Tags [Infra resource] VM snapshot management
// @Accept  json
// @Produce  json
// @Param nsId path string true "Namespace ID" default(ns01)
// @Param mcisId path string true "MCIS ID" default(mcis01)
// @Param vmId path string true "VM ID" default(vm01)
// @Param command path string true "Command to perform" Enums(snapshot)
// @Success 200 {object} mcir.TbCustomImageInfo
// @Failure 404 {object} common.SimpleMsg
// @Failure 500 {object} common.SimpleMsg
// @Router /ns/{nsId}/mcis/{mcisId}/vm/{vmId}/{command} [post]
func RestPostMcisVmWithCmd(c echo.Context) error {

	nsId := c.Param("nsId")
	mcisId := c.Param("mcisId")
	vmId := c.Param("vmId")

	command := strings.Split(c.Path(), "/")[8]
	// c.Path(): /tumblebug/ns/:nsId/mcis/{mcisId}/vm/{vmId}/snapshot

	u := &mcis.TbVmSnapshotReq{}
	if err := c.Bind(u); err != nil {
		return err
	}

	switch command {
	case "snapshot":
		result, err := mcis.CreateVmSnapshot(nsId, mcisId, vmId, u.Name)
		if err != nil {
			mapA := map[string]string{"message": err.Error()}
			return c.JSON(http.StatusNotFound, &mapA)
		}

		// common.PrintJsonPretty(result)

		return c.JSON(http.StatusOK, result)
	default:
		mapA := map[string]string{"message": "Supported commands: snapshot"}
		return c.JSON(http.StatusNotFound, &mapA)
	}
	return nil
}
