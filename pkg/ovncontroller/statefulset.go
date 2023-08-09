/*
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

package ovncontroller

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	ovnv1 "github.com/openstack-k8s-operators/ovn-operator/api/v1beta1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ServiceCommand -
	ServiceCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"

	// PvcSuffixEtcOvn -
	PvcSuffixEtcOvn = "-etc-ovn"
)

// StatefulSet func
func StatefulSet(
	instance *ovnv1.OVNController,
	configHash string,
	labels map[string]string,
	annotations map[string]string,
) *appsv1.StatefulSet {
	runAsUser := int64(0)
	privileged := true

	ovsDbLivenessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      5,
		PeriodSeconds:       3,
		InitialDelaySeconds: 3,
	}
	ovsVswitchdLivenessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      5,
		PeriodSeconds:       3,
		InitialDelaySeconds: 3,
	}

	noopCmd := []string{
		"/bin/true",
	}

	var ovsDbPreStopCmd []string
	var ovsDbCmd []string

	var ovsVswitchdCmd []string
	var ovsVswitchdArgs []string
	var ovsVswitchdPreStopCmd []string

	var ovnControllerArgs []string
	var ovnControllerPreStopCmd []string

	if instance.Spec.Debug.Service {
		ovsDbLivenessProbe.Exec = &corev1.ExecAction{
			Command: noopCmd,
		}
		ovsDbCmd = []string{
			common.DebugCommand,
		}
		ovsDbPreStopCmd = noopCmd

		ovsVswitchdLivenessProbe.Exec = &corev1.ExecAction{
			Command: noopCmd,
		}
		ovsVswitchdCmd = []string{
			common.DebugCommand,
		}
		ovsVswitchdArgs = noopCmd
		ovsVswitchdPreStopCmd = noopCmd

		ovnControllerArgs = []string{
			common.DebugCommand,
		}
		ovnControllerPreStopCmd = noopCmd
	} else {
		ovsDbLivenessProbe.Exec = &corev1.ExecAction{
			Command: []string{
				"/usr/bin/ovs-vsctl",
				"show",
			},
		}
		ovsDbCmd = []string{
			"/usr/local/bin/container-scripts/start-ovsdb-server.sh",
		}
		ovsDbPreStopCmd = []string{
			"/usr/share/openvswitch/scripts/ovs-ctl", "stop", "--no-ovs-vswitchd",
		}

		ovsVswitchdLivenessProbe.Exec = &corev1.ExecAction{
			Command: []string{
				"/usr/bin/ovs-appctl",
				"bond/show",
			},
		}
		ovsVswitchdCmd = []string{
			"/usr/sbin/ovs-vswitchd",
		}
		ovsVswitchdArgs = []string{
			"--pidfile", "--mlockall",
		}
		ovsVswitchdPreStopCmd = []string{
			"/usr/share/openvswitch/scripts/ovs-ctl", "stop", "--no-ovsdb-server",
		}

		ovnControllerArgs = []string{
			"/usr/local/bin/container-scripts/net_setup.sh && ovn-controller --pidfile unix:/run/openvswitch/db.sock",
		}
		ovnControllerPreStopCmd = []string{
			"/usr/share/ovn/scripts/ovn-ctl", "stop_controller",
		}
	}

	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_FILE"] = env.SetValue(KollaConfigAPI)
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["CONFIG_HASH"] = env.SetValue(configHash)

	replicas := int32(3)

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: ServiceName,
			// Replicas:    instance.Spec.Replicas,
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						// ovsdb-server container
						{
							Name:    "ovsdb-server",
							Command: ovsDbCmd,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: ovsDbPreStopCmd,
									},
								},
							},
							Image: instance.Spec.OvsContainerImage,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{"NET_ADMIN", "SYS_ADMIN", "SYS_NICE"},
									Drop: []corev1.Capability{},
								},
								RunAsUser:  &runAsUser,
								Privileged: &privileged,
							},
							Env:                      env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:             GetOvsDbVolumeMounts(),
							Resources:                instance.Spec.Resources,
							LivenessProbe:            ovsDbLivenessProbe,
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
						}, {
							// ovs-vswitchd container
							Name:    "ovs-vswitchd",
							Command: ovsVswitchdCmd,
							Args:    ovsVswitchdArgs,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: ovsVswitchdPreStopCmd,
									},
								},
							},
							Image: instance.Spec.OvsContainerImage,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{"NET_ADMIN", "SYS_ADMIN", "SYS_NICE"},
									Drop: []corev1.Capability{},
								},
								RunAsUser:  &runAsUser,
								Privileged: &privileged,
							},
							Env:                      env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:             GetVswitchdVolumeMounts(),
							Resources:                instance.Spec.Resources,
							LivenessProbe:            ovsVswitchdLivenessProbe,
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
						}, {
							// ovn-controller container
							// NOTE(slaweq): for some reason, when ovn-controller is started without
							// bash shell, it fails with error "unrecognized option --pidfile"
							Name: "ovn-controller",
							Command: []string{
								"/bin/bash", "-c",
							},
							Args: ovnControllerArgs,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: ovnControllerPreStopCmd,
									},
								},
							},
							Image: instance.Spec.OvnContainerImage,
							// TODO(slaweq): to check if ovn-controller really needs such security contexts
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add:  []corev1.Capability{"NET_ADMIN", "SYS_ADMIN", "SYS_NICE"},
									Drop: []corev1.Capability{},
								},
								RunAsUser:  &runAsUser,
								Privileged: &privileged,
							},
							Env:                      env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:             GetOvnControllerVolumeMounts(),
							Resources:                instance.Spec.Resources,
							TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
						},
					},
				},
			},
		},
	}

	// https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#persistentvolumeclaim-retention
	statefulset.Spec.PersistentVolumeClaimRetentionPolicy = &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
		WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
		WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
	}

	blockOwnerDeletion := false
	ownerRef := metav1.NewControllerRef(instance, instance.GroupVersionKind())
	ownerRef.BlockOwnerDeletion = &blockOwnerDeletion

	statefulset.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "var-run",
				Namespace:       instance.Namespace,
				Labels:          labels,
				OwnerReferences: []metav1.OwnerReference{*ownerRef},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				//StorageClassName: &instance.Spec.StorageClass,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						//corev1.ResourceStorage: resource.MustParse(instance.Spec.StorageRequest),
						corev1.ResourceStorage: resource.MustParse("5Gi"),
					},
				},
			},
		},
	}
	statefulset.Spec.Template.Spec.Volumes = GetVolumes(instance.Name)
	// If possible two pods of the same service should not
	// run on the same worker node. If this is not possible
	// the get still created on the same worker node.
	//statefulset.Spec.Template.Spec.Affinity = affinity.DistributePods(
	//	common.AppSelector,
	//	[]string{
	//		serviceName,
	//	},
	//	corev1.LabelHostname,
	//)
	if instance.Spec.NodeSelector != nil && len(instance.Spec.NodeSelector) > 0 {
		statefulset.Spec.Template.Spec.NodeSelector = instance.Spec.NodeSelector
	}

	return statefulset
}
