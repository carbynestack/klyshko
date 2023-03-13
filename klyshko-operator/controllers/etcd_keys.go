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

const rosterKey = "/klyshko/roster"

// Key is a key for data stored in an etcd cluster.
type Key interface {
	ToEtcdKey() string
}

// RosterKey is a Key referencing a set of RosterEntryKey instances. The data referenced by a RosterKey consists of
// information related to a tuple generation job managed mainly by the TupleGenerationJobReconciler.
type RosterKey struct {
	types.NamespacedName
}

// ToEtcdKey converts RosterKey k to an etcd key.
func (k RosterKey) ToEtcdKey() string {
	return fmt.Sprintf("%s/%s/%s", rosterKey, k.Namespace, k.Name)
}

// String returns a string representation of RosterKey k.
func (k RosterKey) String() string {
	return k.ToEtcdKey()
}

// RosterEntryKey is a Key referencing data that is related to a tuple generation task managed mainly by the
// TupleGenerationTaskReconciler.
type RosterEntryKey struct {
	RosterKey
	PlayerID uint
}

// ToEtcdKey converts RosterEntryKey k to an etcd key.
func (k RosterEntryKey) ToEtcdKey() string {
	return fmt.Sprintf("%s/%d", k.RosterKey.ToEtcdKey(), k.PlayerID)
}

// String returns a string representation of RosterEntryKey k.
func (k RosterEntryKey) String() string {
	return k.ToEtcdKey()
}

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

// ParseKey parses string s into a RosterKey or RosterEntryKey. In case s cannot be parsed in any of the those,
// an error is returned.
func ParseKey(s string) (Key, error) {
	parts := etcdKeyParts(s)
	if parts == nil {
		return nil, fmt.Errorf("not a key: %v", s)
	}
	name := types.NamespacedName{Name: parts["jobName"], Namespace: parts["namespace"]}
	playerID, ok := parts["localPlayerID"]
	if !ok {
		return RosterKey{
			name,
		}, nil
	}
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
