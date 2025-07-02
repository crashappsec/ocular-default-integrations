// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package crawlers

import "github.com/crashappsec/ocular/pkg/schemas"

var allCrawlers = map[string]DefaultCrawler{
	"gitlab": {
		Crawler: GitLab{},
		Definition: schemas.Crawler{
			Parameters: map[string]schemas.ParameterDefinition{
				GitLabGroupsParamName: {
					Description: "Comma-separated list of GitLab groups to crawl. If empty or not provided, the crawler will crawl all accessible projects on the instance.",
					Required:    false,
				},
				GitlabInstanceURLParamName: {
					Description: "Base URL of the GitLab instance to crawl. Defaults to 'https://gitlab.com'.",
					Required:    false,
					Default:     "https://gitlab.com",
				},
			},
			UserContainer: schemas.UserContainer{
				Secrets: []schemas.SecretRef{
					{
						Name:        "gitlab-token",
						MountTarget: GitlabTokenSecretEnvVar,
						MountType:   schemas.SecretMountTypeEnvVar,
					},
				},
			},
		},
	},
	"github": {
		Crawler: GitHubOrg{},
		Definition: schemas.Crawler{
			Parameters: map[string]schemas.ParameterDefinition{
				GitHubOrgsParamName: {
					Description: "Comma-separated list of GitLab groups to crawl.",
					Required:    true,
				},
			},
			UserContainer: schemas.UserContainer{
				Secrets: []schemas.SecretRef{
					{
						Name:        "github-token",
						MountTarget: GitHubTokenSecretEnvVar,
						MountType:   schemas.SecretMountTypeEnvVar,
					},
				},
			},
		},
	},
}
