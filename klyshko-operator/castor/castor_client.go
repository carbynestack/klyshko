/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package castor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/carbynestack/klyshko/logging"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client is a client for the Castor tuple store.
type Client struct {
	URL    string
	client *http.Client
}

// NewClient creates a Castor client talking to the given URL.
func NewClient(url string) *Client {
	return &Client{
		URL:    url,
		client: &http.Client{},
	}
}

// ActivateTupleChunk activates the tuple chunk with the given chunk identifier stored by the Castor service.
func (c Client) ActivateTupleChunk(ctx context.Context, chunkID uuid.UUID) error {
	logger := log.FromContext(ctx).WithValues("TupleChunkId", chunkID)
	url := fmt.Sprintf("%s/intra-vcp/tuple-chunks/activate/%s", c.URL, chunkID)
	logger.V(logging.DEBUG).Info("Activating tuple chunk with castor URL", "URL", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received response with status code %d", resp.StatusCode)
	}
	defer func() {
		_, err := io.Copy(ioutil.Discard, resp.Body)
		if err != nil {
			logger.Error(err, "Failed to discard response from castor")
		}
		err = resp.Body.Close()
		if err != nil {
			logger.Error(err, "Failed to close response from castor")
		}
	}()
	logger.V(logging.DEBUG).Info("Response from castor", "Status", resp.Status)
	return nil
}

// TupleMetrics stores how many tuples are available for a given tuple type and how fast they are consumed.
type TupleMetrics struct {
	Available       int    `json:"available"`
	ConsumptionRate int    `json:"consumptionRate"`
	TupleType       string `json:"type"`
}

// Telemetry stores a TupleMetrics object per tuple type.
type Telemetry struct {
	TupleMetrics []TupleMetrics `json:"metrics"`
}

// DeepCopy creates a deep copy of a telemetry struct.
func (t Telemetry) DeepCopy() Telemetry {
	c := Telemetry{
		TupleMetrics: []TupleMetrics{},
	}
	for _, m := range t.TupleMetrics {
		c.TupleMetrics = append(c.TupleMetrics, m)
	}
	return c
}

// GetTelemetry fetches telemetry data from the Castor service.
func (c Client) GetTelemetry(ctx context.Context) (Telemetry, error) {
	logger := log.FromContext(ctx)

	// Building the request
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/intra-vcp/telemetry", c.URL),
		nil,
	)
	if err != nil {
		return Telemetry{}, fmt.Errorf("failed to build request for castor telemetry data")
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	// Doing the request
	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error(err, "Failed to fetch castor telemetry data")
		return Telemetry{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Telemetry{}, fmt.Errorf("received response with status code %d", resp.StatusCode)
	}

	// Read, parse, and return telemetry response
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logger.Error(err, "Failed to close response from castor")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Telemetry{}, fmt.Errorf("failed to read response body")
	}
	var response Telemetry
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return Telemetry{}, err
	}
	return response, nil
}
