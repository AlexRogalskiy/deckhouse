// Copyright 2021 Flant CJSC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*

User-stories:
1. All nodes in cluster have annotation 'node.deckhouse.io/type', hook must group, count them and store to `global.discovery.nodeCountByType`.

*/

package hooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/deckhouse/deckhouse/testing/hooks"
)

var _ = Describe("Global hooks :: discovery/cluster_count_node_types ::", func() {
	const (
		initValuesString       = `{"global": {"discovery": {}}}`
		initConfigValuesString = `{}`
	)

	const (
		stateClusterHasNoTypedNodes = `
apiVersion: v1
kind: Node
metadata:
  name: master
`
		stateClusterHasTypedNodes = `
apiVersion: v1
kind: Node
metadata:
  name: master
  labels:
    node.deckhouse.io/type: Static
---
apiVersion: v1
kind: Node
metadata:
  name: front-1
  labels:
    node.deckhouse.io/type: Cloud
---
apiVersion: v1
kind: Node
metadata:
  name: front-2
  labels:
    node.deckhouse.io/type: Cloud
---
apiVersion: v1
kind: Node
metadata:
  name: system-1
  labels:
    node.deckhouse.io/type: Hybrid
---
apiVersion: v1
kind: Node
metadata:
  name: system-2
  labels:
    node.deckhouse.io/type: Hybrid
`
		stateClusterHasModifiedTypedNodes = `
apiVersion: v1
kind: Node
metadata:
  name: master
  labels:
    node.deckhouse.io/type: Static
---
apiVersion: v1
kind: Node
metadata:
  name: front-1
  labels:
    node.deckhouse.io/type: Cloud
---
apiVersion: v1
kind: Node
metadata:
  name: front-2
  labels:
    node.deckhouse.io/type: Cloud
---
apiVersion: v1
kind: Node
metadata:
  name: system-new-1
  labels:
    node.deckhouse.io/type: Cloud
---
apiVersion: v1
kind: Node
metadata:
  name: system-new-2
  labels:
    node.deckhouse.io/type: Cloud
`
	)

	f := HookExecutionConfigInit(initValuesString, initConfigValuesString)

	Context("Cluster has no typed nodes", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateClusterHasNoTypedNodes))
			f.RunHook()
		})

		It("filterResult of master must be null; `global.discovery.nodeCountByType` must be empty map", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.BindingContexts.Array()).ShouldNot(BeEmpty())
			Expect(len(f.BindingContexts.Get("0.snapshots.node_types").Array())).To(Equal(1))
			Expect(f.BindingContexts.Get("0.snapshots.node_types.0.filterResult").Value()).To(BeEmpty())
			Expect(f.ValuesGet("global.discovery.nodeCountByType").Map()).To(BeEmpty())
		})

		Context("Typed nodes added", func() {
			BeforeEach(func() {
				f.BindingContexts.Set(f.KubeStateSet(stateClusterHasTypedNodes))
				f.RunHook()
			})

			It("`global.discovery.nodeCountByType` must contain map of nodes", func() {
				Expect(f).To(ExecuteSuccessfully())
				Expect(f.ValuesGet("global.discovery.nodeCountByType").String()).To(MatchJSON(`{"static": 1, "cloud": 2, "hybrid": 2}`))
			})

			Context("Nodes modified", func() {
				BeforeEach(func() {
					f.BindingContexts.Set(f.KubeStateSet(stateClusterHasModifiedTypedNodes))
					f.RunHook()
				})

				It("`global.discovery.nodeCountByType` must contain map of nodes", func() {
					Expect(f).To(ExecuteSuccessfully())
					Expect(f.ValuesGet("global.discovery.nodeCountByType").String()).To(MatchJSON(`{"static": 1, "cloud": 4}`))
				})

			})

		})

	})

	Context("Cluster has typed nodes", func() {
		BeforeEach(func() {
			f.BindingContexts.Set(f.KubeStateSet(stateClusterHasTypedNodes))
			f.RunHook()
		})

		It("`global.discovery.nodeCountByType` must contain map of nodes", func() {
			Expect(f).To(ExecuteSuccessfully())
			Expect(f.ValuesGet("global.discovery.nodeCountByType").String()).To(MatchJSON(`{"static": 1, "cloud": 2, "hybrid": 2}`))
		})
	})

})
