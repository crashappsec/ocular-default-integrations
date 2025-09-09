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
}

func GenerateObjects(image string) []*v1beta1.Downloader {
	var downloaders []*v1beta1.Downloader
	for _, c := range AllDownloaders {
		downloaders = append(downloaders, &v1beta1.Downloader{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "Downloader",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.GetName(),
			},
			Spec: v1beta1.DownloaderSpec{
				Container: corev1.Container{
					Name:  c.GetName(),
					Image: image,
				},
			},
		})
	}
	return downloaders
}
