// Copyright (C) 2025 Crash Override, Inc.
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
	"k8s.io/utils/ptr"
)

// CrawledTarget represents a target that has been crawled that should have a pipeline run created for it.
// It includes the target itself and the default downloader to use for it (if not overridden setting
// the Downloader parameter).
type CrawledTarget struct {
	Target            v1beta1.Target
	DefaultDownloader string
}

var AllCrawlers = []Crawler{
	GitHubOrg{},
	GitLab{},
	GHCR{},
	ECR{},
	Dockerhub{},
	StaticList{},
}

const (
	ProfileParamName              = "PROFILE"
	SleepDurationParamName        = "SLEEP_DURATION"
	SleepDurationDefaultValue     = "1m"
	PipelineTTLParamName          = "PIPELINE_TTL"
	PipelineTTLDefaultValue       = "168h" // 7 days
	DownloaderOverrideParamName   = "DOWNLOADER_OVERRIDE"
	ScanServiceAccountParamName   = "SCAN_SERVICE_ACCOUNT"
	UploadServiceAccountParamName = "UPLOAD_SERVICE_ACCOUNT"
)

var DefaultParameters = []v1beta1.ParameterDefinition{
	{
		Name:        ProfileParamName,
		Description: "Profile to use for the pipelines created from this crawler.",
		Required:    true,
	},
	{
		Name:        SleepDurationParamName,
		Description: "Duration to sleep between requests. Will be parsed as a time.Duration.",
		Required:    false,
		Default:     ptr.To(SleepDurationDefaultValue),
	},
	{
		Name:        PipelineTTLParamName,
		Description: "TTL for the pipelines created by this crawler. Will be parsed as a time.Duration.",
		Required:    false,
		Default:     ptr.To(PipelineTTLDefaultValue), // 7 days
	},
	{
		Name:        DownloaderOverrideParamName,
		Description: "Override the downloader for the crawler. By default, it will be chosen based on the crawler type.",
		Required:    false,
	},
	{
		Name:        ScanServiceAccountParamName,
		Description: "Service account to use for the pipelines created from this crawler.",
		Required:    false,
	},
	{
		Name:        UploadServiceAccountParamName,
		Description: "Service account to use for the uploaders in the pipelines created from this crawler.",
		Required:    false,
	},
}

type Crawler interface {
	GetParameters() []v1beta1.ParameterDefinition
	GetName() string
	Crawl(ctx context.Context, params map[string]string, queue chan CrawledTarget) error
	GetEnvSecrets() []definitions.EnvironmentSecret
	GetFileSecrets() []definitions.FileSecret
	EnvironmentVariables() []corev1.EnvVar
}

func GenerateObjects(image, secretName string) []*v1beta1.Crawler {
	crawlerObjs := make([]*v1beta1.Crawler, 0, len(AllCrawlers))
	for _, c := range AllCrawlers {
		crawlerParams := c.GetParameters()

		seenParams := make(map[string]struct{}, len(crawlerParams))
		for _, p := range crawlerParams {
			seenParams[p.Name] = struct{}{}
		}

		// add default parameters to the crawler parameters if they don't already exist
		// this allows us to have common parameters across all crawlers, but crawlers can override them if needed
		for _, v := range DefaultParameters {
			if _, ok := seenParams[v.Name]; !ok {
				crawlerParams = append(crawlerParams, v)
			}
		}

		crawlerObj := &v1beta1.Crawler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "Crawler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: c.GetName(),
			},
			Spec: v1beta1.CrawlerSpec{
				Container: corev1.Container{
					Name:  c.GetName(),
					Image: image,
				},
				Parameters: crawlerParams,
			},
		}

		if envVars := c.EnvironmentVariables(); envVars != nil {
			crawlerObj.Spec.Container.Env = envVars
		}

		if envSecrets := c.GetEnvSecrets(); envSecrets != nil {
			crawlerObj.Spec.Container.Env = definitions.EnvironmentSecretsToEnvVars(secretName, envSecrets)
		}

		if fileSecrets := c.GetFileSecrets(); fileSecrets != nil {
			volume, mounts := definitions.FileSecretsToVolumeMounts(secretName, c.GetName(), fileSecrets)
			crawlerObj.Spec.Volumes = append(crawlerObj.Spec.Volumes, volume)
			crawlerObj.Spec.Container.VolumeMounts = append(crawlerObj.Spec.Container.VolumeMounts, mounts...)
		}

		crawlerObjs = append(crawlerObjs, crawlerObj)
	}
	return crawlerObjs
}
