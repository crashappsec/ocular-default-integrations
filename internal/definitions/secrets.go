// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package definitions

import (
	"github.com/aws/smithy-go/ptr"
	v1 "k8s.io/api/core/v1"
)

type FileSecret struct {
	SecretKey string
	MountPath string
}

type EnvironmentSecret struct {
	SecretKey  string
	EnvVarName string
}

func FileSecretsToVolumeMounts(secretName, resourceName string, secrets []FileSecret) (v1.Volume, []v1.VolumeMount) {
	volumeName := resourceName + "-file-secrets"
	volume := v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
				Optional:   ptr.Bool(true),
			},
		},
	}

	mounts := make([]v1.VolumeMount, 0, len(secrets))
	for _, secret := range secrets {
		mounts = append(mounts, v1.VolumeMount{
			Name:      volume.Name,
			MountPath: secret.MountPath,
			SubPath:   secret.SecretKey,
			ReadOnly:  true,
		})
	}
	return volume, mounts
}

func EnvironmentSecretsToEnvVars(secretName string, secrets []EnvironmentSecret) []v1.EnvVar {
	envVars := make([]v1.EnvVar, 0, len(secrets))
	for _, secret := range secrets {
		envVars = append(envVars, v1.EnvVar{
			Name: secret.EnvVarName,
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					Key:      secret.SecretKey,
					Optional: ptr.Bool(true),
					LocalObjectReference: v1.LocalObjectReference{
						Name: secretName,
					},
				},
			},
		})
	}
	return envVars
}
