package ovncontroller

import (
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(name string) []corev1.Volume {
	var scriptsVolumeDefaultMode int32 = 0755
	return []corev1.Volume{
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
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
		{
			Name:      "var-run",
			MountPath: "/var/run",
			ReadOnly:  false,
		},
	}
}

// GetVswitchdVolumeMounts - ovs-vswitchd VolumeMounts
func GetVswitchdVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "var-run",
			MountPath: "/var/run",
			ReadOnly:  false,
		},
	}
}

// GetOvnControllerVolumeMounts - ovn-controller VolumeMounts
func GetOvnControllerVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
		{
			Name:      "var-run",
			MountPath: "/var/run",
			ReadOnly:  false,
		},
	}
}
