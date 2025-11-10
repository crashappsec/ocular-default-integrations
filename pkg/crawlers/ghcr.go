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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-github/v71/github"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RecentTagLimitParam = "RECENT_TAG_LIMIT"
)

type GHCR struct{}

func (g GHCR) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        GitHubOrgsParamName,
			Description: "Comma-separated list of Docker Hub organizations to crawl.",
			Required:    true,
		},
		{
			Name: RecentTagLimitParam,
			Description: "Maximum number of tags (versions) to retrieve per image. " +
				"Will retrieve the latest N tags for each GHCR image and start a new pipeline for each. " +
				"Set to 0 to retrieve all versions. Defaults to 1.",
			Required: false,
			Default:  ptr.To("1"),
		},
	}
}

func (g GHCR) GetName() string {
	return "ghcr"
}

func (g GHCR) Crawl(ctx context.Context, params map[string]string, queue chan CrawledTarget) error {
	l := log.FromContext(ctx).WithValues("crawler", "ghcr")
	// retrieve params
	orgs := strings.Split(params[GitHubOrgsParamName], ",")
	token := os.Getenv(GitHubTokenSecretEnvVar)
	downloader := downloaders.Docker{}.GetName()

	limit, err := strconv.Atoi(params[RecentTagLimitParam])
	if err != nil {
		l.Error(err, "invalid value for limit, defaulting to 1", "limit", RecentTagLimitParam)
		limit = 1
	}

	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no github org specified")
	}

	var merr *multierror.Error
	for _, org := range orgs {
		// check if org is org or user
		user, _, err := client.Users.Get(ctx, org)
		if err != nil {
			l.Error(err, "error retrieving org info", "org", org)
			merr = multierror.Append(merr, err)
		}
		if user.GetType() == "Organization" {
			err = crawlGHCRContainers(ctx, org, downloader, queue, client.Organizations, limit)
		} else {
			err = crawlGHCRContainers(ctx, org, downloader, queue, client.Users, limit)
		}
		if err != nil {
			l.Error(err, "Error crawling org", "org", org)
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}

type GHCRPackgeIndexer interface {
	ListPackages(
		context.Context, string, *github.PackageListOptions,
	) ([]*github.Package, *github.Response, error)
	PackageGetAllVersions(
		context.Context, string, string, string, *github.PackageListOptions,
	) ([]*github.PackageVersion, *github.Response, error)
}

func crawlGHCRContainers(
	ctx context.Context,
	org, dl string,
	queue chan CrawledTarget,
	indexer GHCRPackgeIndexer,
	tagLimit int,
) error {
	l := log.FromContext(ctx)

	opt := &github.ListOptions{PerPage: 100}
	var merr *multierror.Error
	for {
		containers, _, err := indexer.ListPackages(ctx, org, &github.PackageListOptions{
			PackageType: github.Ptr("container"),
			ListOptions: *opt,
		})
		if err != nil {
			merr = multierror.Append(merr, err)
			break
		}
		for _, container := range containers {
			versions, err := getRecentGHCRTags(ctx, org, container.GetName(), indexer, tagLimit)
			if err != nil {
				merr = multierror.Append(merr, err)
				break
			}
			targetID := fmt.Sprintf("ghcr.io/%s/%s", org, container.GetName())
			for _, version := range versions {
				target := CrawledTarget{
					Target: v1beta1.Target{
						Identifier: targetID,
						Version:    version,
					},
					DefaultDownloader: dl,
				}
				l.Info("Discovered GHCR container", "identifier", targetID, "version", version)
				queue <- target
			}
		}
	}
	return merr.ErrorOrNil()
}

func getRecentGHCRTags(ctx context.Context,
	org string,
	packageName string,
	indexer GHCRPackgeIndexer,
	tagLimit int,
) ([]string, error) {
	var version []string
	opt := &github.ListOptions{PerPage: 100}
	for {
		versions, resp, err := indexer.PackageGetAllVersions(ctx, org, "container", packageName, &github.PackageListOptions{
			ListOptions: *opt,
		})
		if err != nil {
			return nil, err
		}
		for _, v := range versions {
			metadata, ok := v.GetMetadata()
			if ok {
				if dockerMetadata := metadata.GetContainer(); dockerMetadata != nil {
					if tagList := dockerMetadata.Tags; len(tagList) > 0 {
						// only get the first tag as version, since all other tags
						// in list point to the same version
						version = append(version, tagList[0])
					}
				}
			}
		}
		if resp.NextPage == 0 || (tagLimit > 0 && len(version) >= tagLimit) {
			break
		}
	}

	if tagLimit > 0 && len(version) > tagLimit {
		version = version[:tagLimit]
	}
	return version, nil
}

func (g GHCR) GetEnvSecrets() []definitions.EnvironmentSecret {
	return []definitions.EnvironmentSecret{
		{
			SecretKey:  "github-token",
			EnvVarName: GitHubTokenSecretEnvVar,
		},
	}
}

func (g GHCR) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (g GHCR) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

var _ Crawler = GHCR{}
