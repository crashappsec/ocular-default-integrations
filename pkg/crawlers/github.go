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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/internal/utils"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-github/v71/github"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GitHubOrg struct{}

var _ Crawler = GitHubOrg{}

func (GitHubOrg) GetName() string {
	return "github"
}

const (
	GitHubTokenSecretEnvVar = "GITHUB_TOKEN"
	GitHubAppInstallationID = "GITHUB_APP_INSTALLATION_ID"
	GitHubAppID             = "GITHUB_APP_ID"
	GitHubAppPrivateKey     = "GITHUB_APP_PRIVATE_KEY"
)

func (o GitHubOrg) GetEnvSecrets() []definitions.EnvironmentSecret {
	return []definitions.EnvironmentSecret{
		{
			SecretKey:  "github-token",
			EnvVarName: GitHubTokenSecretEnvVar,
		},
		{
			SecretKey:  "github-app-installation-id",
			EnvVarName: GitHubAppInstallationID,
		},
		{
			SecretKey:  "github-app-private-key",
			EnvVarName: GitHubAppPrivateKey,
		},
		{
			SecretKey:  "github-app-id",
			EnvVarName: GitHubAppID,
		},
	}
}

func (o GitHubOrg) GetFileSecrets() []definitions.FileSecret {
	return nil
}
func (o GitHubOrg) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

const (
	GitHubOrgsParamName      = "GITHUB_ORGS"
	GitHubSkipForksParamName = "SKIP_FORKS"
)

func (GitHubOrg) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        GitHubOrgsParamName,
			Description: "Comma-separated list of GitLab groups to crawl.",
			Required:    true,
		},
		{
			Name:        GitHubSkipForksParamName,
			Description: "If set to anything but '0' or 'false', forked repositories will be skipped.",
			Required:    false,
			Default:     ptr.To("false"),
		},
	}
}

// Crawl retrieves all repositories from a specified GitHub organization
// and sends their clone URLs to the provided queue channel. By default, the downloader
// used is "git", but this can be overridden by setting the parameter variable
// [DownloaderParamName] to a different value. The GitHub token can be
// set by setting the secret [GithubTokenParam].
func (GitHubOrg) Crawl(
	baseCtx context.Context,
	params map[string]string,
	queue chan CrawledTarget,
) error {
	l := log.FromContext(baseCtx).WithValues("crawler", "github")
	ctx := log.IntoContext(baseCtx, l)
	// retrieve params
	orgs := strings.Split(params[GitHubOrgsParamName], ",")
	skipForksParam := strings.ToLower(params[GitHubSkipForksParamName])
	skipForks := skipForksParam != "" && skipForksParam != "0" && skipForksParam != "false"
	downloader := downloaders.Git{}.GetName()

	client := createGitHubClient(ctx)
	if len(orgs) == 0 {
		return fmt.Errorf("no github org specified")
	}

	var merr *multierror.Error
	for _, org := range orgs {
		if err := crawlOrg(ctx, client, org, downloader, skipForks, queue); err != nil {
			l.Error(err, "Error crawling org", "org", org)
			merr = multierror.Append(merr, err)
		}
	}

	return merr.ErrorOrNil()
}

func createGitHubClient(ctx context.Context) *github.Client {
	l := log.FromContext(ctx)
	installationIDStr := os.Getenv(GitHubAppInstallationID)
	appIDStr := os.Getenv(GitHubAppID)
	privateKey := os.Getenv(GitHubAppPrivateKey)
	token := os.Getenv(GitHubTokenSecretEnvVar)

	if installationIDStr != "" && privateKey != "" && appIDStr != "" {
		l.Info("authenticating using GitHub App")
		appID, appIDErr := strconv.ParseInt(appIDStr, 10, 64)
		if appIDErr != nil {
			l.Error(appIDErr, "failed to parse GitHub App ID")
		}
		installationID, installationIDErr := strconv.ParseInt(installationIDStr, 10, 64)
		if installationIDErr != nil {
			l.Error(installationIDErr, "failed to parse GitHub Installation ID")
		}
		if installationIDErr == nil && appIDErr == nil {
			// both parsed successfully, if not fall through to token auth
			itr, err := utils.AuthenticateGitHubApp(ctx, appID, installationID, []byte(privateKey))
			if err != nil {
				l.Error(err, "failed to authenticate GitHub App")
			} else {
				return github.NewClient(&http.Client{Transport: itr})
			}
		}
	}

	if token != "" {
		l.Info("authenticating using GitHub Token")
		return github.NewClient(nil).WithAuthToken(token)
	}

	l.Info("no GitHub authentication configured, proceeding unauthenticated")
	return github.NewClient(nil)
}

func crawlOrg(
	ctx context.Context,
	c *github.Client,
	org, dl string,
	skipForks bool,
	queue chan CrawledTarget,
) error {
	l := log.FromContext(ctx)

	l.Info("crawling github org", "org", org)
	// check if org is org or user
	user, _, err := c.Users.Get(ctx, org)
	if err != nil {
		return fmt.Errorf("error getting org info: %w", err)
	}

	isOrg := user.GetType() == "Organization"
	l = l.WithValues(
		"org", org,
		"org_type", user.GetType())

	l.Info("beginning to crawl github repositories")
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
			if skipForks && repo.GetFork() {
				l.Info("skipping forked repository", "repo", repo.GetFullName())
				continue
			}
			l.Info("enqueuing repository", "repo", repo.GetFullName(), "url", repo.GetCloneURL())
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
				l.
					Error(convertErr, "unable to convert ratelimit reset", "reset", reset)
				l.Info("using default sleep duration", "duration", sleep)
			} else {
				sleep = time.Until(time.Unix(int64(resetTime), 0))
			}
			l.Info("rate limit reached, sleeping until reset", "duration", sleep)
			time.Sleep(sleep)
		}

		opt.Page = resp.NextPage
	}
	l.Info("crawling complete")
	return nil
}
