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

const VcpConfigMapName = "cs-vcp-config"

func getVcpConfig(ctx context.Context, client *client.Client, namespace string) (v1.ConfigMap, error) {
	name := types.NamespacedName{
		Namespace: namespace,
		Name:      VcpConfigMapName,
	}
	cfm := v1.ConfigMap{}
	err := (*client).Get(ctx, name, &cfm)
	if err != nil {
		return cfm, errors.Unwrap(fmt.Errorf("VCP configuration not found: %w", err))
	}
	return cfm, nil
}

func localPlayerID(ctx context.Context, client *client.Client, namespace string) (uint, error) {
	cfm, err := getVcpConfig(ctx, client, namespace)
	if err != nil {
		return 0, err
	}

	// Extract playerId
	playerIDStr, ok := cfm.Data["playerId"]
	if !ok {
		return 0, errors.New("invalid VCP configuration - missing playerId")
	}
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		return 0, err
	}
	if playerID < 0 || playerID > math.MaxUint32 {
		return 0, fmt.Errorf("invalid playerId '%d'- must be in range [0,%d]", playerID, math.MaxUint32)
	}
	return uint(playerID), nil
}

func numberOfPlayers(ctx context.Context, client *client.Client, namespace string) (uint, error) {
	cfm, err := getVcpConfig(ctx, client, namespace)
	if err != nil {
		return 0, err
	}

	// Extract playerCount
	playerCountStr, ok := cfm.Data["playerCount"]
	if !ok {
		return 0, errors.New("invalid VCP configuration - missing playerCount")
	}
	playerCount, err := strconv.Atoi(playerCountStr)
	if err != nil {
		return 0, err
	}
	if playerCount < 0 || playerCount > math.MaxUint32 {
		return 0, fmt.Errorf("invalid playerCount '%d'- must be in range [0,%d]", playerCount, math.MaxUint32)
	}
	return uint(playerCount), nil
}
