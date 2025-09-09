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
	"time"

	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-github/v71/github"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type GitHubOrg struct{}

var _ Crawler = GitHubOrg{}

func (GitHubOrg) GetName() string {
	return "github"
}

/**************
 * Parameters *
 **************/

const (
	GitHubOrgsParamName = "GITHUB_ORGS"
)

func (GitHubOrg) GetParameters() map[string]v1beta1.ParameterDefinition {
	return map[string]v1beta1.ParameterDefinition{
		GitHubOrgsParamName: {
			Description: "Comma-separated list of GitLab groups to crawl.",
			Required:    true,
		},
	}
}

/************
 * Secrets  *
 ************/

const (
	GitHubTokenSecretEnvVar = "GITLAB_TOKEN"
)

// Crawl retrieves all repositories from a specified GitHub organization
// and sends their clone URLs to the provided queue channel. By default, the downloader
// used is "git", but this can be overridden by setting the parameter variable
// [DownloaderParamName] to a different value. The GitHub token can be
// set by setting the secret [GithubTokenParam].
func (GitHubOrg) Crawl(
	ctx context.Context,
	params map[string]string,
	queue chan CrawledTarget,
) error {
	// retrieve params
	orgs := strings.Split(params[GitHubOrgsParamName], ",")
	token := os.Getenv(GitHubTokenSecretEnvVar)
	downloader := downloaders.Git{}.GetName()

	client := github.NewClient(nil)
	if token != "" {
		client = client.WithAuthToken(token)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no github org specified")
	}

	var merr *multierror.Error
	for _, org := range orgs {
		if err := crawlOrg(ctx, client, org, downloader, queue); err != nil {
			zap.L().Error("Error crawling org", zap.String("org", org), zap.Error(err))
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}

func crawlOrg(
	ctx context.Context,
	c *github.Client,
	org, dl string,
	queue chan CrawledTarget,
) error {
	// check if org is org or user
	user, _, err := c.Users.Get(ctx, org)
	if err != nil {
		return fmt.Errorf("error getting org info: %w", err)
	}

	isOrg := user.GetType() == "Organization"

	opt := github.ListOptions{PerPage: 100}
	for {
		var (
			repos []*github.Repository
			resp  *github.Response
		)
		if isOrg {
			repos, resp, err = c.Repositories.ListByOrg(
				ctx,
				org,
				&github.RepositoryListByOrgOptions{
					ListOptions: opt,
				},
			)
		} else {
			repos, resp, err = c.Repositories.ListByUser(ctx, org, &github.RepositoryListByUserOptions{
				ListOptions: opt,
			})
		}
		if err != nil {
			return err
		}

		for _, repo := range repos {
			queue <- CrawledTarget{
				Target: v1beta1.Target{
					Identifier: repo.GetCloneURL(),
				},
				DefaultDownloader: dl,
			}
		}
		if resp.NextPage == 0 {
			break
		}

		// Attempt to handle rate limiting via header
		if strings.TrimSpace(resp.Header.Get("x-ratelimit-remaining")) == "0" {
			reset := resp.Header.Get("x-ratelimit-reset")
			resetTime, convertErr := strconv.Atoi(reset)
			sleep := time.Hour
			if convertErr != nil {
				zap.L().
					Error("unable to convert ratelimit reset", zap.String("reset", reset), zap.Error(convertErr))
				zap.L().Info("using default sleep duration", zap.Duration("duration", sleep))
			} else {
				sleep = time.Until(time.Unix(int64(resetTime), 0))
			}
			zap.L().
				Info("rate limit reached, sleeping until reset", zap.Duration("duration", sleep))
			time.Sleep(sleep)
		}

		opt.Page = resp.NextPage
	}
	return nil
}
