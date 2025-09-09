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
	GetParameters() map[string]v1beta1.ParameterDefinition
	GetName() string
	Upload(
		ctx context.Context,
		metadata input.PipelineMetadata,
		params map[string]string,
		files []string,
	) error
}

func GenerateObjects(image string) []*v1beta1.Uploader {
	var uploaders []*v1beta1.Uploader
	for _, c := range AllUploaders {
		params := c.GetParameters()
		uploaders = append(uploaders, &v1beta1.Uploader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "Uploader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.GetName(),
			},
			Spec: v1beta1.UploaderSpec{
				Container: corev1.Container{
					Name:  c.GetName(),
					Image: image,
				},
				Parameters: params,
			},
		})
	}
	return uploaders
}
