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
	"github.com/crashappsec/ocular/pkg/schemas"
	"github.com/hashicorp/go-multierror"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/zap"
)

type GitLab struct{}

/**************
 * Parameters *
 **************/

const (
	GitLabGroupsParamName          = "GITLAB_GROUPS"
	GitlabInstanceURLParamName     = "GITLAB_INSTANCE_URL"
	GitlabIncludeSubgroupParamName = "INCLUDE_SUBGROUPS"
)

/************
 * Secrets  *
 ************/

const (
	GitlabTokenSecretEnvVar = "GITLAB_TOKEN"
)

// Crawl retrieves all repositories from a specified GitLab groups
// and sends their clone URLs to the provided queue channel. By default, the downloader
// used is "git", but this can be overridden by setting the parameter variable
// [DownloaderParamName] to a different value.
func (g GitLab) Crawl(
	ctx context.Context,
	params map[string]string,
	queue chan schemas.Target,
) error {
	groups := strings.Split(params[GitLabGroupsParamName], ",")
	token := os.Getenv(GitlabTokenSecretEnvVar)

	baseURL := params[GitlabInstanceURLParamName]

	// Check if the recursive parameter is set
	includeSubGroup := params[GitlabIncludeSubgroupParamName] != ""

	// will default to use default git downloader.
	// This will be overridden in the main function if 'DOWNLOADER' param is set
	downloader := downloaders.GitDownloaderName

	l := zap.L().
		With(zap.String("url", baseURL), zap.String("downloader", downloader), zap.Strings("groups", groups))

	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return fmt.Errorf("error creating gitlab client: %w", err)
	}
	if len(groups) == 0 {
		// if there are no groups specified, crawl the entire instance
		return crawlGitlabInstance(ctx, client, downloader, queue)
	}

	var merr *multierror.Error
	l.Info(fmt.Sprintf("crawling %d gitlab groups", len(groups)), zap.Int("groups", len(groups)))
	for _, group := range groups {
		groupL := l.With(zap.String("group", group))
		groupL.Debug(fmt.Sprintf("crawling gitlab group %s", group), zap.String("group", group))
		if err := crawlGitlabGroup(ctx, client, group, downloader, includeSubGroup, queue); err != nil {
			groupL.Error("Error crawling gitlab group", zap.String("group", group), zap.Error(err))
			merr = multierror.Append(merr, err)
		}
	}
	l.Info("finished crawling gitlab groups", zap.Int("groups", len(groups)))

	return merr.ErrorOrNil()
}

func crawlGitlabGroup(
	_ context.Context,
	c *gitlab.Client,
	org, dl string, includeSubGroups bool,
	queue chan schemas.Target,
) error {
	opt := gitlab.ListOptions{PerPage: 100}
	for {
		var (
			projs []*gitlab.Project
			resp  *gitlab.Response
			err   error
		)

		projs, resp, err = c.Groups.ListGroupProjects(
			org,
			&gitlab.ListGroupProjectsOptions{
				ListOptions:      opt,
				IncludeSubGroups: &includeSubGroups,
			},
		)
		if err != nil {
			return err
		}

		for _, repo := range projs {
			queue <- schemas.Target{
				Identifier: repo.HTTPURLToRepo,
				Downloader: dl,
			}
		}
		if resp.NextPage == 0 || resp.NextPage >= resp.TotalPages {
			break
		}

		// Attempt to handle rate limiting via header
		if strings.TrimSpace(resp.Header.Get("RateLimit-Remaining")) == "0" {
			reset := resp.Header.Get("RateLimit-Reset")
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

func crawlGitlabInstance(
	ctx context.Context,
	c *gitlab.Client,
	dl string,
	queue chan schemas.Target,
) error {
	opt := gitlab.ListOptions{PerPage: 100}
	for {
		var (
			groups []*gitlab.Group
			resp   *gitlab.Response
			err    error
		)

		groups, resp, err = c.Groups.ListGroups(
			&gitlab.ListGroupsOptions{
				ListOptions: opt,
			},
		)
		if err != nil {
			return err
		}

		for _, group := range groups {
			err = crawlGitlabGroup(ctx, c, group.FullPath, dl, true, queue)
			if err != nil {
				zap.L().
					Error("Error crawling gitlab group", zap.String("group", group.FullPath), zap.Error(err))
				continue
			}
		}
		if resp.NextPage == 0 || resp.NextPage >= resp.TotalPages {
			break
		}

		// Attempt to handle rate limiting via header
		if strings.TrimSpace(resp.Header.Get("RateLimit-Remaining")) == "0" {
			reset := resp.Header.Get("RateLimit-Reset")
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
