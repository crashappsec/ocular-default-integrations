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

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GitLab struct{}

func (g GitLab) GetName() string {
	return "gitlab"
}

var _ Crawler = GitLab{}

func (GitLab) GetEnvSecrets() []definitions.EnvironmentSecret {
	return []definitions.EnvironmentSecret{
		{
			SecretKey:  "gitlab-token",
			EnvVarName: GitlabTokenSecretEnvVar,
		},
	}
}

func (GitLab) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (GitLab) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

const (
	GitLabGroupsParamName          = "GITLAB_GROUPS"
	GitlabInstanceURLParamName     = "GITLAB_INSTANCE_URL"
	GitlabIncludeSubgroupParamName = "INCLUDE_SUBGROUPS"
)

func (g GitLab) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        GitLabGroupsParamName,
			Description: "Comma-separated list of GitLab groups to crawl. If empty, the entire instance will be crawled.",
			Required:    false,
		},
		{
			Name:        GitlabInstanceURLParamName,
			Description: "The base URL of the GitLab instance to crawl. For GitLab.com, use https://gitlab.com/api/v4",
			Required:    true,
		},
		{
			Name:        GitlabIncludeSubgroupParamName,
			Description: "If set, include projects from subgroups of the specified groups.",
			Required:    false,
		},
	}
}

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
	queue chan CrawledTarget,
) error {
	l := log.FromContext(ctx)
	splitGroups := strings.Split(params[GitLabGroupsParamName], ",")
	var groups []string
	for _, group := range splitGroups {
		trimmed := strings.TrimSpace(group)
		if trimmed != "" {
			groups = append(groups, trimmed)
		}
	}
	l.Info("crawling groups", "groups", groups)

	token := os.Getenv(GitlabTokenSecretEnvVar)

	baseURL := params[GitlabInstanceURLParamName]

	// Check if the recursive parameter is set
	includeSubGroup := params[GitlabIncludeSubgroupParamName] != ""

	// will default to use default git downloader.
	// This will be overridden in the main function if 'DOWNLOADER_OVERRIDE' param is set
	downloader := downloaders.Git{}.GetName()

	l = l.WithValues("url", baseURL, "downloader", downloader, "groups", groups)

	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return fmt.Errorf("error creating gitlab client: %w", err)
	}
	if len(groups) == 0 {
		// if there are no groups specified, crawl the entire instance
		return crawlGitlabInstance(ctx, client, downloader, queue)
	}

	var merr *multierror.Error
	l.Info(fmt.Sprintf("crawling %d gitlab groups", len(groups)), "groups", len(groups))
	for _, group := range groups {
		groupL := l.WithValues("group", group)
		groupL.Info(fmt.Sprintf("crawling gitlab group %s", group))
		if err := crawlGitlabGroup(ctx, client, group, downloader, includeSubGroup, queue); err != nil {
			groupL.Error(err, "Error crawling gitlab group")
			merr = multierror.Append(merr, err)
		}
	}
	l.Info("finished crawling gitlab groups", "groups", len(groups))

	return merr.ErrorOrNil()
}

func crawlGitlabGroup(
	ctx context.Context,
	c *gitlab.Client,
	org, dl string, includeSubGroups bool,
	queue chan CrawledTarget,
) error {
	l := log.FromContext(ctx)
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
			l.Info("enqueuing gitlab repo", "repo", repo.HTTPURLToRepo)
			queue <- CrawledTarget{
				Target: v1beta1.Target{
					Identifier: repo.HTTPURLToRepo,
				},
				DefaultDownloader: dl,
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
				l.Error(convertErr, "unable to convert ratelimit reset", "reset", reset)
				l.Info("using default sleep duration", "duration", sleep)
			} else {
				sleep = time.Until(time.Unix(int64(resetTime), 0))
			}
			l.Info("rate limit reached, sleeping until reset", "duration", sleep)
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
	queue chan CrawledTarget,
) error {
	l := log.FromContext(ctx)
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
				l.Error(err, "Error crawling gitlab group", "group", group.FullPath)
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
				l.Error(convertErr, "unable to convert ratelimit reset", "reset", reset)
				l.Info("using default sleep duration", "duration", sleep)
			} else {
				sleep = time.Until(time.Unix(int64(resetTime), 0))
			}
			l.Info("rate limit reached, sleeping until reset", "duration", sleep)
			time.Sleep(sleep)
		}

		opt.Page = resp.NextPage
	}

	return nil
}
