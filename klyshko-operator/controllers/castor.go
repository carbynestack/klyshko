/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func activateTupleChunk(ctx context.Context, chunkID uuid.UUID) error {
	logger := log.FromContext(ctx).WithValues("TupleChunkId", chunkID)
	client := &http.Client{}
	url := fmt.Sprintf("http://cs-castor.default.svc.cluster.local:10100/intra-vcp/tuple-chunks/activate/%s", chunkID) // TODO Make servername configurable / use discovery
	logger.Info("activating tuple chunk with castor URL", "URL", url)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	logger.Info("response from castor", "Status", resp.Status, "Body", body) // TODO Body is Base64 encoded :-(
	return err
}

type TupleMetrics struct {
	Available       int    `json:"available"`
	ConsumptionRate int    `json:"consumptionRate"`
	TupleType       string `json:"type"`
}

type Telemetry struct {
	TupleMetrics []TupleMetrics `json:"metrics"`
}

func getTelemetry(ctx context.Context) (Telemetry, error) {
	logger := log.FromContext(ctx)
	client := &http.Client{}

	// Building the request
	req, err := http.NewRequest("GET", "http://cs-castor.default.svc.cluster.local:10100/intra-vcp/telemetry", nil) // TODO Make servername configurable / use discovery
	if err != nil {
		logger.Error(err, "failed to build request for castor telemetry data")
		return Telemetry{}, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// Doing the request
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(err, "failed to fetch castor telemetry data")
		return Telemetry{}, err
	}

	// Read and parse response
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "failed to read response body")
		return Telemetry{}, err
	}
	var response Telemetry
	json.Unmarshal(bodyBytes, &response)

	return response, nil
}
