// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"context"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var AllDownloaders = []Downloader{
	Git{},
	docker{},
	gcs{},
	npm{},
	pypi{},
	s3{},
}

type Downloader interface {
	GetName() string
	Download(ctx context.Context, cloneURL, version, targetDir string) error
	GetEnvSecrets() []definitions.EnvironmentSecret
	GetFileSecrets() []definitions.FileSecret
	EnvironmentVariables() []corev1.EnvVar
}

func GenerateObjects(image string) []*v1beta1.Downloader {
	downloaderObjs := make([]*v1beta1.Downloader, 0, len(AllDownloaders))
	for _, d := range AllDownloaders {
		downlaoderObj := &v1beta1.Downloader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "Downloader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: d.GetName(),
			},
			Spec: v1beta1.DownloaderSpec{
				Container: corev1.Container{
					Name:  d.GetName(),
					Image: image,
					Env:   d.EnvironmentVariables(),
				},
			},
		}

		if envVars := d.EnvironmentVariables(); envVars != nil {
			downlaoderObj.Spec.Container.Env = envVars
		}

		if envSecrets := d.GetEnvSecrets(); envSecrets != nil {
			downlaoderObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars("downloaders", envSecrets)
		}

		if fileSecrets := d.GetFileSecrets(); fileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts("downloaders", d.GetName(), fileSecrets)
			downlaoderObj.Spec.Volumes = append(downlaoderObj.Spec.Volumes, volume)
			downlaoderObj.Spec.Container.VolumeMounts = append(downlaoderObj.Spec.Container.VolumeMounts, mounts...)
		}
		downloaderObjs = append(downloaderObjs, downlaoderObj)
	}
	return downloaderObjs
}
