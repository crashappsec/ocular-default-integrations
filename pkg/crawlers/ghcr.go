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
	"strconv"
	"strings"

	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-github/v71/github"
	"github.com/hashicorp/go-multierror"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RecentTagLimitParam = "RECENT_TAG_LIMIT"
)

func init() {
	All.registerCrawler(GHCR)
}

var GHCR = Crawler{
	Name: "ghcr",
	Parameters: []v1beta1.ParameterDefinition{
		{
			Name:        GitHubOrgsParamName,
			Description: "Comma-separated list of Docker Hub organizations to crawl.",
		},
		{
			Name: RecentTagLimitParam,
			Description: "Maximum number of tags (versions) to retrieve per image. " +
				"Will retrieve the latest N tags for each GHCR image and start a new pipeline for each. " +
				"Set to 0 to retrieve all versions. Defaults to 1.",
			Default: ptr.To("1"),
		},
	},
	EnvironmentSecrets: githubAuthenticationEnvironmentSecrets,
	Crawl:              crawlGHCR,
}

func crawlGHCR(baseCtx context.Context, params map[string]string, queue chan v1beta1.Target) error {
	l := log.FromContext(baseCtx).WithValues("crawler", "ghcr")
	ctx := log.IntoContext(baseCtx, l)

	// retrieve params
	orgs := strings.Split(params[GitHubOrgsParamName], ",")

	l.Info("starting GHCR org crawler", "orgs", orgs)
	if len(orgs) == 0 {
		return fmt.Errorf("no GHCR orgs specified")
	}

	limit, err := strconv.Atoi(params[RecentTagLimitParam])
	if err != nil {
		l.Error(err, "invalid value for limit, defaulting to 1", "limit", RecentTagLimitParam)
		limit = 1
	}

	var merr *multierror.Error
	for _, org := range orgs {
		client := createGitHubClientForOrg(ctx, org)
		isUser, err := isGitHubUser(ctx, client, org)
		if err != nil {
			l.Error(err, "Error determining if org is an organization or user", "org", org)
			merr = multierror.Append(merr, err)
			continue
		}
		var indexer GHCRPackageIndexer = client.Organizations
		if isUser {
			indexer = client.Users
		}
		err = crawlGHCRContainers(ctx, org, queue, indexer, limit)
		if err != nil {
			l.Error(err, "Error crawling org", "org", org)
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}

type GHCRPackageIndexer interface {
	ListPackages(
		context.Context, string, *github.PackageListOptions,
	) ([]*github.Package, *github.Response, error)
	PackageGetAllVersions(
		context.Context, string, string, string, *github.PackageListOptions,
	) ([]*github.PackageVersion, *github.Response, error)
}

func crawlGHCRContainers(
	ctx context.Context,
	org string,
	queue chan v1beta1.Target,
	indexer GHCRPackageIndexer,
	tagLimit int,
) error {
	l := log.FromContext(ctx)

	opt := &github.ListOptions{PerPage: 100}
	containers, _, err := indexer.ListPackages(ctx, org, &github.PackageListOptions{
		PackageType: github.Ptr("container"),
		ListOptions: *opt,
	})
	if err != nil {
		return fmt.Errorf("listing GHCR packages for org %q: %v", org, err)
	}

	var merr *multierror.Error
	for _, container := range containers {
		versions, err := getRecentGHCRTags(ctx, org, container.GetName(), indexer, tagLimit)
		if err != nil {
			merr = multierror.Append(merr, err)
			l.Error(err, "Error getting recent tags for container", "container", container.GetName())
			continue
		}
		targetID := fmt.Sprintf("ghcr.io/%s/%s", org, container.GetName())
		for _, version := range versions {
			target := v1beta1.Target{
				Identifier: targetID,
				Version:    version,
			}
			l.Info("Discovered GHCR container", "identifier", targetID, "version", version)
			queue <- target
		}
	}
	return merr.ErrorOrNil()
}

func getRecentGHCRTags(ctx context.Context,
	org string,
	packageName string,
	indexer GHCRPackageIndexer,
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
