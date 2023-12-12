package ovncontroller

import (
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(name string, namespace string) []corev1.Volume {

	var scriptsVolumeDefaultMode int32 = 0755

	//source_type := corev1.HostPathDirectoryOrCreate
	return []corev1.Volume{
		{
			Name: "var-run",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "ovn-controller-pvc0",
				},
			},
		},
		{
			Name: "var-lib",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "ovn-controller-pvc0",
				},
			},
		},
		{
			Name: "scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-scripts",
					},
				},
			},
		},
	}

}

// GetOvsDbVolumeMounts - ovsdb-server VolumeMounts
func GetOvsDbVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "var-run",
			MountPath: "/var/run/openvswitch",
			ReadOnly:  false,
		},
		{
			Name:      "var-lib",
			MountPath: "/var/lib/openvswitch",
			ReadOnly:  false,
		},
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
	}
}

// GetVswitchdVolumeMounts - ovs-vswitchd VolumeMounts
func GetVswitchdVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "var-run",
			MountPath: "/var/run/openvswitch",
			ReadOnly:  false,
		},
		{
			Name:      "var-lib",
			MountPath: "/var/lib/openvswitch",
			ReadOnly:  false,
		},
	}
}

// GetOvnControllerVolumeMounts - ovn-controller VolumeMounts
func GetOvnControllerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "var-run",
			MountPath: "/var/run/openvswitch",
			ReadOnly:  false,
		},
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
	}
}
