/*
Copyright (c) 2022 - for information on the respective copyright owner
see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"math"
)

var _ = When("Parsing a key", func() {
	rosterKey := RosterKey{
		NamespacedName: types.NamespacedName{Namespace: "foo", Name: "bar"},
	}
	rosterEntryKey := RosterEntryKey{
		RosterKey: rosterKey,
		PlayerID:  1,
	}

	Context("from a serialized roster key", func() {
		It("should return original key", func() {
			Expect(ParseKey(rosterKey.ToEtcdKey())).To(Equal(rosterKey))
		})
	})

	Context("from a serialized roster entry key", func() {
		It("should return original key", func() {
			Expect(ParseKey(rosterEntryKey.ToEtcdKey())).To(Equal(rosterEntryKey))
		})
	})

	Context("from a roster entry key with non-integer playerId", func() {
		It("should fail", func() {
			malformedKey := rosterEntryKey.ToEtcdKey() + "m"
			_, err := ParseKey(malformedKey)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("from a roster entry key with non-unit32 playerId", func() {
		It("should fail", func() {
			malformedKey := rosterEntryKey.ToEtcdKey() + fmt.Sprint(math.MaxUint32*2)
			_, err := ParseKey(malformedKey)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("from a roster entry key with non-atoi parseable playerId", func() {
		It("should fail", func() {
			malformedKey := rosterEntryKey.ToEtcdKey() + fmt.Sprint(math.MaxUint64/2)
			_, err := ParseKey(malformedKey)
			Expect(err).To(HaveOccurred())
		})
	})

})
