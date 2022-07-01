/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CastorClient struct {
	Url    string
	client *http.Client
}

func NewCastorClient(url string) *CastorClient {
	return &CastorClient{
		Url:    url,
		client: &http.Client{},
	}
}

func (castorClient CastorClient) activateTupleChunk(ctx context.Context, chunkID uuid.UUID) error {
	logger := log.FromContext(ctx).WithValues("TupleChunkId", chunkID)
	url := fmt.Sprintf("%s/intra-vcp/tuple-chunks/activate/%s", castorClient.Url, chunkID)
	logger.Info("activating tuple chunk with castor URL", "URL", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	resp, err := castorClient.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("received response with status code %d", resp.StatusCode))
	}
	defer func() {
		_, err := io.Copy(ioutil.Discard, resp.Body)
		if err != nil {
			logger.Error(err, "failed to discard response from castor")
		}
		err = resp.Body.Close()
		if err != nil {
			logger.Error(err, "failed to close response from castor")
		}
	}()
	logger.Info("response from castor", "Status", resp.Status)
	return nil
}

type TupleMetrics struct {
	Available       int    `json:"available"`
	ConsumptionRate int    `json:"consumptionRate"`
	TupleType       string `json:"type"`
}

type Telemetry struct {
	TupleMetrics []TupleMetrics `json:"metrics"`
}

func (castorClient CastorClient) getTelemetry(ctx context.Context) (Telemetry, error) {
	logger := log.FromContext(ctx)

	// Building the request
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/intra-vcp/telemetry", castorClient.Url),
		nil,
	)
	if err != nil {
		logger.Error(err, "failed to build request for castor telemetry data")
		return Telemetry{}, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// Doing the request
	resp, err := castorClient.client.Do(req)
	if err != nil {
		logger.Error(err, "failed to fetch castor telemetry data")
		return Telemetry{}, err
	}
	if resp.StatusCode != http.StatusOK {
		err := errors.New(fmt.Sprintf("received response with status code %d", resp.StatusCode))
		logger.Error(err, "failed to fetch castor telemetry data")
		return Telemetry{}, err
	}

	// Read, parse, and return telemetry response
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logger.Error(err, "failed to close response from castor")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err, "failed to read response body")
		return Telemetry{}, err
	}
	var response Telemetry
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return Telemetry{}, err
	}
	return response, nil
}
