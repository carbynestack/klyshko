/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"math"
	"regexp"
	"strconv"
)

type Key interface {
	ToEtcdKey() string
}

type RosterKey struct {
	types.NamespacedName
}

func (k RosterKey) ToEtcdKey() string {
	return rosterKey + "/" + k.Namespace + "/" + k.Name
}

func (k RosterKey) String() string {
	return k.ToEtcdKey()
}

type RosterEntryKey struct {
	RosterKey
	PlayerID uint
}

func (k RosterEntryKey) ToEtcdKey() string {
	return rosterKey + "/" + k.Namespace + "/" + k.Name + "/" + strconv.Itoa(int(k.PlayerID))
}

func (k RosterEntryKey) String() string {
	return k.ToEtcdKey()
}

const rosterKey = "/klyshko/roster"

var etcdRosterKeyPattern = regexp.MustCompile("^" + rosterKey + "/(?P<namespace>(\\w|-)+)/(?P<jobName>(\\w|-)+)(?:/(?P<localPlayerID>\\d+))?$")

func etcdKeyParts(s string) map[string]string {
	match := etcdRosterKeyPattern.FindStringSubmatch(s)
	if match == nil {
		return nil
	}
	result := make(map[string]string)
	for i, name := range etcdRosterKeyPattern.SubexpNames() {
		if i != 0 && name != "" && match[i] != "" {
			result[name] = match[i]
		}
	}
	return result
}

func ParseKey(s string) (Key, error) {
	parts := etcdKeyParts(s)
	if parts == nil {
		return nil, fmt.Errorf("not a key: %v", s)
	}
	name := types.NamespacedName{Name: parts["jobName"], Namespace: parts["namespace"]}
	if playerID, ok := parts["localPlayerID"]; ok {
		pid, err := strconv.Atoi(playerID)
		if err != nil {
			return nil, fmt.Errorf("invalid playerId '%v' - not an integer: %w", playerID, err)
		}
		if pid < 0 || pid > math.MaxUint32 {
			return nil, fmt.Errorf("invalid playerId '%d' - must be in range [0,%d]", pid, math.MaxUint32)
		}
		return RosterEntryKey{
			RosterKey: RosterKey{
				name,
			},
			PlayerID: uint(pid),
		}, nil
	}
	return RosterKey{
		name,
	}, nil
}
