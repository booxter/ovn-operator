/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package functional_test

import (
	"encoding/json"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

var _ = Describe("OVNNorthd controller", func() {

	When("A OVNNorthd instance is created", func() {
		var ovnNorthdName types.NamespacedName
		BeforeEach(func() {
			ovnNorthdName = ovn.CreateOVNNorthd(namespace, GetDefaultOVNNorthdSpec())
			DeferCleanup(ovn.DeleteOVNNorthd, ovnNorthdName)
		})

		It("should have the Spec fields initialized", func() {
			OVNNorthd := ovn.GetOVNNorthd(ovnNorthdName)
			Expect(*(OVNNorthd.Spec.Replicas)).Should(Equal(int32(1)))
		})

		It("should have the Status fields initialized", func() {
			OVNNorthd := ovn.GetOVNNorthd(ovnNorthdName)
			Expect(OVNNorthd.Status.ReadyCount).To(Equal(int32(0)))
		})

		It("should have a finalizer", func() {
			// the reconciler loop adds the finalizer so we have to wait for
			// it to run
			Eventually(func() []string {
				return ovn.GetOVNNorthd(ovnNorthdName).Finalizers
			}, timeout, interval).Should(ContainElement("OVNNorthd"))
		})

		It("should be in input ready condition", func() {
			th.ExpectCondition(
				ovnNorthdName,
				ConditionGetterFunc(OVNNorthdConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
		})

		When("OVNDBCluster instances are available", func() {
			It("should create a Deployment with the ovn connection CLI args set based on the OVNDBCluster", func() {
				dbs := CreateOVNDBClusters(namespace, map[string][]string{}, 1)
				DeferCleanup(DeleteOVNDBClusters, dbs)

				deplName := types.NamespacedName{
					Namespace: namespace,
					Name:      "ovn-northd",
				}

				depl := th.GetDeployment(deplName)
				Expect(depl.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{
					"-vfile:off", "-vconsole:info",
					"--ovnnb-db=tcp:ovsdbserver-nb-0." + namespace + ".svc.cluster.local:6641",
					"--ovnsb-db=tcp:ovsdbserver-sb-0." + namespace + ".svc.cluster.local:6642",
				}))
			})
		})

	})

	When("A OVNNorthd instance is created with debug on", func() {
		BeforeEach(func() {
			dbs := CreateOVNDBClusters(namespace, map[string][]string{}, 1)
			DeferCleanup(DeleteOVNDBClusters, dbs)
			spec := GetDefaultOVNNorthdSpec()
			spec.Debug.Service = true
			ovnNorthdName := ovn.CreateOVNNorthd(namespace, spec)
			DeferCleanup(ovn.DeleteOVNNorthd, ovnNorthdName)
		})

		It("Container commands to include debug commands", func() {
			deplName := types.NamespacedName{
				Namespace: namespace,
				Name:      "ovn-northd",
			}

			depl := th.GetDeployment(deplName)
			Expect(depl.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(depl.Spec.Template.Spec.Containers[0].LivenessProbe.Exec.Command).To(
				Equal([]string{"/bin/true"}))
			Expect(depl.Spec.Template.Spec.Containers[0].ReadinessProbe.Exec.Command).To(
				Equal([]string{"/bin/true"}))
			Expect(depl.Spec.Template.Spec.Containers[0].Command[0]).Should(ContainSubstring("/bin/sleep"))
			Expect(depl.Spec.Template.Spec.Containers[0].Args[0]).Should(ContainSubstring("infinity"))
		})
	})

	When("OVNNorthd is created with networkAttachments", func() {
		var ovnNorthdName types.NamespacedName

		BeforeEach(func() {
			dbs := CreateOVNDBClusters(namespace, map[string][]string{}, 1)
			DeferCleanup(DeleteOVNDBClusters, dbs)
			spec := GetDefaultOVNNorthdSpec()
			spec.NetworkAttachment = "internalapi"
			ovnNorthdName = ovn.CreateOVNNorthd(namespace, spec)
			DeferCleanup(ovn.DeleteOVNNorthd, ovnNorthdName)
		})

		It("reports that the definition is missing", func() {
			th.ExpectConditionWithDetails(
				ovnNorthdName,
				ConditionGetterFunc(OVNNorthdConditionGetter),
				condition.NetworkAttachmentsReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				"NetworkAttachment resources missing: internalapi",
			)
		})
		It("reports that network attachment is missing", func() {
			internalAPINADName := types.NamespacedName{Namespace: namespace, Name: "internalapi"}
			nad := th.CreateNetworkAttachmentDefinition(internalAPINADName)
			DeferCleanup(th.DeleteInstance, nad)

			deplName := types.NamespacedName{
				Namespace: namespace,
				Name:      "ovn-northd",
			}

			depl := th.GetDeployment(deplName)

			expectedAnnotation, err := json.Marshal(
				[]networkv1.NetworkSelectionElement{
					{
						Name:             "internalapi",
						Namespace:        namespace,
						InterfaceRequest: "internalapi",
					}})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(depl.Spec.Template.ObjectMeta.Annotations).To(
				HaveKeyWithValue("k8s.v1.cni.cncf.io/networks", string(expectedAnnotation)),
			)

			// We don't add network attachment status annotations to the Pods
			// to simulate that the network attachments are missing.
			//th.SimulateDeploymentReadyWithPods(deplName, map[string][]string{})

			th.ExpectConditionWithDetails(
				ovnNorthdName,
				ConditionGetterFunc(OVNNorthdConditionGetter),
				condition.NetworkAttachmentsReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				"NetworkAttachments error occurred "+
					"not all pods have interfaces with ips as configured in NetworkAttachments: internalapi",
			)
		})
		It("reports that an IP is missing", func() {
			internalAPINADName := types.NamespacedName{Namespace: namespace, Name: "internalapi"}
			nad := th.CreateNetworkAttachmentDefinition(internalAPINADName)
			DeferCleanup(th.DeleteInstance, nad)

			deplName := types.NamespacedName{
				Namespace: namespace,
				Name:      "ovn-northd",
			}

			depl := th.GetDeployment(deplName)

			expectedAnnotation, err := json.Marshal(
				[]networkv1.NetworkSelectionElement{
					{
						Name:             "internalapi",
						Namespace:        namespace,
						InterfaceRequest: "internalapi",
					}})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(depl.Spec.Template.ObjectMeta.Annotations).To(
				HaveKeyWithValue("k8s.v1.cni.cncf.io/networks", string(expectedAnnotation)),
			)

			// We simulate that there is no IP associated with the internalapi
			// network attachment
			th.SimulateDeploymentReadyWithPods(
				deplName,
				map[string][]string{namespace + "/internalapi": {}},
			)

			th.ExpectConditionWithDetails(
				ovnNorthdName,
				ConditionGetterFunc(OVNNorthdConditionGetter),
				condition.NetworkAttachmentsReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				"NetworkAttachments error occurred "+
					"not all pods have interfaces with ips as configured in NetworkAttachments: internalapi",
			)
		})
		It("reports NetworkAttachmentsReady if the Pods got the proper annotations", func() {
			internalAPINADName := types.NamespacedName{Namespace: namespace, Name: "internalapi"}
			nad := th.CreateNetworkAttachmentDefinition(internalAPINADName)
			DeferCleanup(th.DeleteInstance, nad)

			deplName := types.NamespacedName{
				Namespace: namespace,
				Name:      "ovn-northd",
			}

			th.SimulateDeploymentReadyWithPods(
				deplName,
				map[string][]string{namespace + "/internalapi": {"10.0.0.1"}},
			)

			th.ExpectCondition(
				ovnNorthdName,
				ConditionGetterFunc(OVNNorthdConditionGetter),
				condition.NetworkAttachmentsReadyCondition,
				corev1.ConditionTrue,
			)

			Eventually(func(g Gomega) {
				ovnNorthd := GetOVNNorthd(ovnNorthdName)
				g.Expect(ovnNorthd.Status.NetworkAttachments).To(
					Equal(map[string][]string{namespace + "/internalapi": {"10.0.0.1"}}))

			}, timeout, interval).Should(Succeed())
		})
	})

})
