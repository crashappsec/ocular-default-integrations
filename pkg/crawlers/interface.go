// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package crawlers

import (
	"context"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var All = make(all)

type all map[string]Crawler

func (a all) registerCrawler(c Crawler) {
	if a == nil {
		return
	}
	if c.Crawl == nil {
		panic("crawl function must be set")
	}
	a[c.Name] = c
}

type Crawler struct {
	Name                 string
	Parameters           []v1beta1.ParameterDefinition
	Crawl                func(ctx context.Context, params map[string]string, queue chan v1beta1.Target) error
	EnvironmentSecrets   []definitions.EnvironmentSecret
	FileSecrets          []definitions.FileSecret
	EnviornmentVariables []corev1.EnvVar
}

func GenerateObjects(image, secretName string) []*v1beta1.ClusterCrawler {
	crawlerObjs := make([]*v1beta1.ClusterCrawler, 0, len(All))
	for _, c := range All {
		crawlerParams := c.Parameters

		seenParams := make(map[string]struct{}, len(crawlerParams))
		for _, p := range crawlerParams {
			seenParams[p.Name] = struct{}{}
		}

		crawlerObj := &v1beta1.ClusterCrawler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "ClusterCrawler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.Name,
			},
			Spec: v1beta1.CrawlerSpec{
				Container: corev1.Container{
					Name:  c.Name,
					Image: image,
					Env:   c.EnviornmentVariables,
				},
				Parameters: crawlerParams,
			},
		}

		if envSecrets := c.EnvironmentSecrets; envSecrets != nil {
			crawlerObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars(secretName, envSecrets)
		}

		if fileSecrets := c.FileSecrets; fileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts(secretName, c.Name, fileSecrets)
			crawlerObj.Spec.Volumes = append(crawlerObj.Spec.Volumes, volume)
			crawlerObj.Spec.Container.VolumeMounts = append(crawlerObj.Spec.Container.VolumeMounts, mounts...)
		}

		crawlerObjs = append(crawlerObjs, crawlerObj)
	}
	return crawlerObjs
}
