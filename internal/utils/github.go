// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package utils

import (
	"context"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v71/github"
)

func AuthenticateGitHubAppInstallation(_ context.Context, appID, installationID int64, privatePEM []byte) (*ghinstallation.Transport, error) {
	// Shared transport to reuse TCP connections.
	tr := http.DefaultTransport
	// Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
	itr, err := ghinstallation.New(tr, appID, installationID, privatePEM)
	if err != nil {
		log.Fatal(err)
	}
	return itr, nil
}

func AuthenticateGitHubAppForRepository(ctx context.Context, org, repo string, appID int64, privatePEM []byte) (*ghinstallation.Transport, error) {
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privatePEM)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(&http.Client{Transport: transport})

	installation, _, err := ghClient.Apps.FindRepositoryInstallation(ctx, org, repo)
	if err != nil {
		return nil, err
	}
	return AuthenticateGitHubAppInstallation(ctx, appID, installation.GetID(), privatePEM)
}

func AuthenticateGitHubAppForOrg(ctx context.Context, org string, appID int64, privatePEM []byte) (*ghinstallation.Transport, error) {
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privatePEM)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(&http.Client{Transport: transport})
	installation, _, err := ghClient.Apps.FindOrganizationInstallation(ctx, org)
	if err != nil {
		return nil, err
	}
	return AuthenticateGitHubAppInstallation(ctx, appID, installation.GetID(), privatePEM)
}

func AuthenticateGitHubAppForUser(ctx context.Context, user string, appID int64, privatePEM []byte) (*ghinstallation.Transport, error) {
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privatePEM)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(&http.Client{Transport: transport})
	installation, _, err := ghClient.Apps.FindUserInstallation(ctx, user)
	if err != nil {
		return nil, err
	}
	return AuthenticateGitHubAppInstallation(ctx, appID, installation.GetID(), privatePEM)
}
