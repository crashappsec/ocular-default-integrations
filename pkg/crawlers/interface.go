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

	"github.com/crashappsec/ocular/pkg/schemas"
)

func GetAllDefaults() map[string]DefaultCrawler {
	result := make(map[string]DefaultCrawler, len(allCrawlers))
	for name, def := range allCrawlers {
		for paramName, paramDef := range defaultParameters {
			if _, exists := def.Definition.Parameters[paramName]; !exists {
				def.Definition.Parameters[paramName] = paramDef
			}
		}
		result[name] = def
	}
	return result
}

type Crawler interface {
	Crawl(ctx context.Context, params map[string]string, queue chan schemas.Target) error
}

type DefaultCrawler struct {
	Definition schemas.Crawler
	Crawler    Crawler `json:"crawler"`
}

const (
	ProfileParamName       = "PROFILE"
	SleepDurationParamName = "SLEEP_DURATION"
	DownloaderParamName    = "DOWNLOADER"
)

var defaultParameters = map[string]schemas.ParameterDefinition{
	ProfileParamName: {
		Description: "Profile to use for the crawler.",
		Required:    true,
	},
	SleepDurationParamName: {
		Description: "Duration to sleep between requests. Will be parsed as a time.Duration.",
		Required:    false,
		Default:     "1m",
	},
	DownloaderParamName: {
		Description: "Override the downloader for the crawler. The default will be chosen based on the crawler type.",
		Required:    false,
	},
}
