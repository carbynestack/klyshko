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

const vcpConfigMapName = "cs-vcp-config"

func getVCPConfig(ctx context.Context, client *client.Client, namespace string) (v1.ConfigMap, error) {
	name := types.NamespacedName{
		Namespace: namespace,
		Name:      vcpConfigMapName,
	}
	cfm := v1.ConfigMap{}
	err := (*client).Get(ctx, name, &cfm)
	if err != nil {
		return cfm, errors.Unwrap(fmt.Errorf("VCP configuration not found: %w", err))
	}
	return cfm, nil
}

func parseVCPConfig(ctx context.Context, client *client.Client, namespace string) (uint, uint, error) {
	cfm, err := getVCPConfig(ctx, client, namespace)
	if err != nil {
		return 0, 0, err
	}

	// Extract playerCount
	playerCountStr, ok := cfm.Data["playerCount"]
	if !ok {
		return 0, 0, errors.New("invalid VCP configuration - missing playerCount")
	}
	playerCount, err := strconv.Atoi(playerCountStr)
	if err != nil {
		return 0, 0, err
	}
	if playerCount < 0 || playerCount > math.MaxUint32 {
		return 0, 0, fmt.Errorf("invalid playerCount '%d' - must be in range [0,%d]", playerCount, math.MaxUint32)
	}

	// Extract playerId
	playerIDStr, ok := cfm.Data["playerId"]
	if !ok {
		return 0, 0, errors.New("invalid VCP configuration - missing playerId")
	}
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		return 0, 0, err
	}
	if playerID < 0 || playerID >= playerCount {
		return 0, 0, fmt.Errorf("invalid playerId '%d' - must be in range [0,%d]", playerID, playerCount)
	}

	return uint(playerID), uint(playerCount), nil
}

func localPlayerID(ctx context.Context, client *client.Client, namespace string) (uint, error) {
	playerID, _, err := parseVCPConfig(ctx, client, namespace)
	if err != nil {
		return 0, err
	}
	return playerID, nil
}

func numberOfVCPs(ctx context.Context, client *client.Client, namespace string) (uint, error) {
	_, playerCount, err := parseVCPConfig(ctx, client, namespace)
	if err != nil {
		return 0, err
	}
	return playerCount, nil
}
