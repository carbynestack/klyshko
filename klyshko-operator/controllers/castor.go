/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func activateTupleChunk(ctx context.Context, chunkId uuid.UUID) error {
	logger := log.FromContext(ctx).WithValues("TupleChunkId", chunkId)
	client := &http.Client{}
	url := fmt.Sprintf("http://cs-castor.default.svc.cluster.local:10100/intra-vcp/tuple-chunks/activate/%s", chunkId) // TODO Make servername configurable / use discovery
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
