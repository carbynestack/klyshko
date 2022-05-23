/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"math"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

func localPlayerID(ctx context.Context, client *client.Client, namespace string) (uint, error) {

	// Get VCP configuration config map
	name := types.NamespacedName{
		Namespace: namespace,
		Name:      "vcp-config",
	}
	cfm := &v1.ConfigMap{}
	err := (*client).Get(ctx, name, cfm)
	if err != nil {
		return 0, errors.Unwrap(fmt.Errorf("VCP configuration not found: %w", err))
	}

	// Extract playerId
	if playerID, ok := cfm.Data["playerId"]; ok {
		pid, err := strconv.Atoi(playerID)
		if err != nil {
			return 0, err
		}
		if pid < 0 || pid > math.MaxUint32 {
			return 0, fmt.Errorf("invalid playerId '%d'- must be in range [0,%d]", pid, math.MaxUint32)
		}
		return uint(pid), nil
	} else {
		return 0, errors.New("invalid VCP configuration - missing playerId")
	}
}
