// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"context"

	"github.com/crashappsec/ocular/pkg/schemas"
)

type Downloader interface {
	Download(ctx context.Context, cloneURL, version, targetDir string) error
}

type DefaultDownloader struct {
	Definition schemas.Downloader
	Downloader Downloader
}

func GetAllDefaults() map[string]DefaultDownloader {
	result := make(map[string]DefaultDownloader, len(allDownloaders))
	for name, def := range allDownloaders {
		result[name] = def
	}
	return result
}
