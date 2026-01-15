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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/clients/dockerhub"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DockerHubOrgsParam = "DOCKERHUB_ORGS"

	DockerHubTokenSecretEnvVar = "DOCKERHUB_TOKEN"
)

type Dockerhub struct{}

func (d Dockerhub) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        DockerHubOrgsParam,
			Description: "Comma-separated list of Docker Hub organizations to crawl.",
			Required:    true,
		},
		{
			Name: RecentTagLimitParam,
			Description: "Maximum number of tags (versions) to retrieve per image. " +
				"Will retrieve the latest N tags for each docker hub image and start a new pipeline for each. " +
				"Set to 0 to retrieve all versions. Defaults to 1.",
			Required: false,
			Default:  ptr.To("1"),
		},
	}
}

func (d Dockerhub) GetName() string {
	return "dockerhub"
}

func (d Dockerhub) Crawl(ctx context.Context, params map[string]string, queue chan CrawledTarget) error {
	l := log.FromContext(ctx).WithValues("crawler", "dockerhub")
	// retrieve params
	orgs := strings.Split(params[DockerHubOrgsParam], ",")
	token := os.Getenv(DockerHubTokenSecretEnvVar)
	downloader := downloaders.Docker{}.GetName()

	client := dockerhub.NewClient(dockerhub.Options{
		AuthToken: token,
	})

	limit, err := strconv.Atoi(params[RecentTagLimitParam])
	if err != nil {
		l.Error(err, "invalid value for limit, defaulting to 1", "limit", RecentTagLimitParam)
		limit = 1
	}

	if len(orgs) == 0 {
		return fmt.Errorf("no dockerhub org specified")
	}

	var merr *multierror.Error
	for _, o := range orgs {
		org := strings.TrimSpace(o)
		// check if org is org or user
		repositories, err := client.ListNamespaceRepositories(ctx, org)
		if err != nil {
			l.Error(err, "error retrieving org info", "org", org)
			merr = multierror.Append(merr, err)
			continue
		}
		for _, repo := range repositories {
			repoName := fmt.Sprintf("docker.io/%s/%s", org, repo.Name)
			tags, err := client.ListRepositoryTags(ctx, org, repo.Name)
			if err != nil {
				l.Error(err, "error retrieving tags", "repository", repoName)
				merr = multierror.Append(merr, err)
				continue
			}
			if limit > 0 && len(tags) > limit {
				tags = tags[:limit]
			}
			for _, tag := range tags {
				targetVersion := tag.Name
				l.Info("queuing target", "repository", repoName, "tag", targetVersion)
				queue <- CrawledTarget{
					DefaultDownloader: downloader,
					Target: v1beta1.Target{
						Version:    targetVersion,
						Identifier: repoName,
					},
				}
			}
		}
	}

	return merr.ErrorOrNil()
}

func (d Dockerhub) GetEnvSecrets() []definitions.EnvironmentSecret {
	return []definitions.EnvironmentSecret{
		{
			SecretKey:  "dockerhub-token",
			EnvVarName: DockerHubTokenSecretEnvVar,
		},
	}
}

func (d Dockerhub) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (d Dockerhub) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

var _ Crawler = Dockerhub{}
