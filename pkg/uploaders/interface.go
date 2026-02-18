// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package uploaders

import (
	"context"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var All = make(all)

type all map[string]Uploader

func (a all) registerUploader(u Uploader) {
	if a == nil {
		return
	}
	if u.Upload == nil {
		panic("upload function must be set")
	}
	a[u.Name] = u
}

type Uploader struct {
	Name       string
	Parameters []v1beta1.ParameterDefinition
	Upload     func(
		ctx context.Context,
		metadata input.PipelineMetadata,
		params map[string]string,
		files []string,
	) error
	EnvironmentSecrets   []definitions.EnvironmentSecret
	FileSecrets          []definitions.FileSecret
	EnvironmentVariables []corev1.EnvVar
}

func GenerateObjects(image, secretName string) []*v1beta1.ClusterUploader {
	uploaderObjs := make([]*v1beta1.ClusterUploader, 0, len(All))
	for _, u := range All {
		params := u.Parameters
		uploaderObj := &v1beta1.ClusterUploader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "ClusterUploader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: u.Name,
			},
			Spec: v1beta1.UploaderSpec{
				Container: corev1.Container{
					Name:  u.Name,
					Image: image,
					Env:   u.EnvironmentVariables,
				},
				Parameters: params,
			},
		}

		if u.EnvironmentSecrets != nil {
			uploaderObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars(secretName, u.EnvironmentSecrets)
		}

		if u.FileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts(secretName, u.Name, u.FileSecrets)
			uploaderObj.Spec.Volumes = append(uploaderObj.Spec.Volumes, volume)
			uploaderObj.Spec.Container.VolumeMounts = append(uploaderObj.Spec.Container.VolumeMounts, mounts...)
		}
		uploaderObjs = append(uploaderObjs, uploaderObj)
	}
	return uploaderObjs
}
