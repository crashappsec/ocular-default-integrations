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

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/clients/aws"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ECR struct{}

func (e ECR) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (e ECR) GetParameters() []v1beta1.ParameterDefinition {
	return append(aws.GetParameters(),
		v1beta1.ParameterDefinition{
			Name: RecentTagLimitParam,
			Description: "Maximum number of tags (versions) to retrieve per image. " +
				"Will retrieve the latest N tags for each ECR image and start a new pipeline for each. " +
				"Set to 0 to retrieve all versions. Defaults to 1.",
			Required: false,
			Default:  ptr.To("1"),
		})
}

func (e ECR) GetName() string {
	return "ecr"
}

func (e ECR) Crawl(ctx context.Context, params map[string]string, queue chan CrawledTarget) error {
	l := log.FromContext(ctx).WithValues("crawler", "ghcr")
	regionOverride := params[aws.RegionParamName]
	profileOverride := params[aws.ProfileParamName]
	recentTagLimit, err := strconv.Atoi(params[RecentTagLimitParam])
	if err != nil {
		l.Error(err, "invalid recent tag limit parameter, defaulting to 1")
		recentTagLimit = 1
	}

	cfg, err := aws.BuildConfig(ctx, aws.WithProfile(profileOverride), aws.WithRegionOverride(regionOverride))
	if err != nil {
		l.Error(err, "Failed to load AWS configuration")
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if regionOverride != "" {
		cfg.Region = regionOverride
	}

	ecrClient := ecr.NewFromConfig(cfg)
	var merr *multierror.Error
	output, err := ecrClient.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
	if err != nil {
		l.Error(err, "error describing ECR repositories")
		return fmt.Errorf("error describing ECR repositories: %w", err)
	}

	for _, repo := range output.Repositories {
		repoName := *repo.RepositoryName
		tagsOutput, err := ecrClient.ListTagsForResource(ctx, &ecr.ListTagsForResourceInput{
			ResourceArn: repo.RepositoryArn,
		})
		if err != nil {
			l.Error(err, "error listing tags for repository", "repository", repoName)
			merr = multierror.Append(merr, err)
			continue
		}

		// For simplicity, assuming tagsOutput.Tags contains the tags.
		tags := tagsOutput.Tags
		if recentTagLimit > 0 && len(tags) > recentTagLimit {
			tags = tags[:recentTagLimit]
		}
		for _, tag := range tags {
			targetVersion := *tag.Key
			l.Info("queuing target", "repository", repoName, "tag", targetVersion)
			queue <- CrawledTarget{
				DefaultDownloader: downloaders.Docker{}.GetName(),
				Target: v1beta1.Target{
					Version:    targetVersion,
					Identifier: fmt.Sprintf("%s/%s", *repo.RepositoryUri, repoName),
				},
			}
		}

	}
	return merr.ErrorOrNil()
}

func (e ECR) GetFileSecrets() []definitions.FileSecret {
	return aws.GetAWSFileSecrets()
}

func (e ECR) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

var _ Crawler = ECR{}
