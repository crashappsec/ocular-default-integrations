// Copyright (C) 2025 Crash Override, Inc.
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
)

func AuthenticateGitHubApp(_ context.Context, appID, installationID int64, privatePEM []byte) (*ghinstallation.Transport, error) {
	// Shared transport to reuse TCP connections.
	tr := http.DefaultTransport
	// Wrap the shared transport for use with the app ID 1 authenticating with installation ID 99.
	itr, err := ghinstallation.New(tr, appID, installationID, privatePEM)
	if err != nil {
		log.Fatal(err)
	}
	return itr, nil
}
