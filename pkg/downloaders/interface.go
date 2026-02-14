// Copyright (C) 2025-2026 Crash Override, Inc.
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

var All = make(all)

type all map[string]Downloader

func (a all) registerDownloader(d Downloader) {
	if a == nil {
		return
	}

	if d.Download == nil {
		panic("download function must be defined")
	}

	a[d.Name] = d
}

type Downloader struct {
	Parameters           []v1beta1.ParameterDefinition
	Name                 string
	Download             func(ctx context.Context, params map[string]string, identifier, version, targetDir string) error
	EnvironmentSecrets   []definitions.EnvironmentSecret
	FileSecrets          []definitions.FileSecret
	EnvironmentVariables []corev1.EnvVar
	MetadataFiles        []string
}

func GenerateObjects(image, secretName string) []*v1beta1.ClusterDownloader {
	downloaderObjs := make([]*v1beta1.ClusterDownloader, 0, len(All))
	for _, d := range All {
		downlaoderObj := &v1beta1.ClusterDownloader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "ClusterDownloader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: d.Name,
			},
			Spec: v1beta1.DownloaderSpec{
				Container: corev1.Container{
					Name:  d.Name,
					Image: image,
					Env:   d.EnvironmentVariables,
				},
				MetadataFiles: d.MetadataFiles,
				Parameters:    d.Parameters,
			},
		}
		if d.EnvironmentSecrets != nil {
			downlaoderObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars(secretName, d.EnvironmentSecrets)
		}

		if d.FileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts(secretName, d.Name, d.FileSecrets)
			downlaoderObj.Spec.Volumes = append(downlaoderObj.Spec.Volumes, volume)
			downlaoderObj.Spec.Container.VolumeMounts = append(downlaoderObj.Spec.Container.VolumeMounts, mounts...)
		}
		downloaderObjs = append(downloaderObjs, downlaoderObj)
	}
	return downloaderObjs
}
