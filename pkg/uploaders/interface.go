// Copyright (C) 2025 Crash Override, Inc.
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

var AllUploaders = []Uploader{
	s3{},
	webhook{},
}

type Uploader interface {
	GetParameters() []v1beta1.ParameterDefinition
	GetName() string
	Upload(
		ctx context.Context,
		metadata input.PipelineMetadata,
		params map[string]string,
		files []string,
	) error
	GetEnvSecrets() []definitions.EnvironmentSecret
	GetFileSecrets() []definitions.FileSecret
	EnvironmentVariables() []corev1.EnvVar
}

func GenerateObjects(image string) []*v1beta1.Uploader {
	uploaderObjs := make([]*v1beta1.Uploader, 0, len(AllUploaders))
	for _, u := range AllUploaders {
		params := u.GetParameters()
		uploaderObj := &v1beta1.Uploader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "Uploader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: u.GetName(),
			},
			Spec: v1beta1.UploaderSpec{
				Container: corev1.Container{
					Name:  u.GetName(),
					Image: image,
				},
				Parameters: params,
			},
		}

		if envVars := u.EnvironmentVariables(); envVars != nil {
			uploaderObj.Spec.Container.Env = envVars
		}

		if envSecrets := u.GetEnvSecrets(); envSecrets != nil {
			uploaderObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars("uploaders", envSecrets)
		}

		if fileSecrets := u.GetFileSecrets(); fileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts("uploaders", u.GetName(), fileSecrets)
			uploaderObj.Spec.Volumes = append(uploaderObj.Spec.Volumes, volume)
			uploaderObj.Spec.Container.VolumeMounts = append(uploaderObj.Spec.Container.VolumeMounts, mounts...)
		}
		uploaderObjs = append(uploaderObjs, uploaderObj)
	}
	return uploaderObjs
}
