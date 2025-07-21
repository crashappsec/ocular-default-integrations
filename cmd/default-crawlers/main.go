// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

// Default crawlers bundled with Ocular.
// This is intended to run multiple crawlers depending on the value of the
// environment variable OCULAR_CRAWLER_NAME.  For more infomration, See the [crawlers] package for more details.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/crashappsec/ocular-default-integrations/internal/config"
	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	apiClient "github.com/crashappsec/ocular/pkg/api/client"
	"github.com/crashappsec/ocular/pkg/schemas"
	"go.uber.org/zap"
)

func init() {
	config.Init()
}

func main() {
	ctx := context.Background()
	crawlerName := os.Getenv(schemas.EnvVarCrawlerName)
	if crawlerName == "" {
		zap.L().Fatal("no crawler specified")
	}

	apiBaseURL := os.Getenv(schemas.EnvVarAPIBaseURL)
	if apiBaseURL == "" {
		zap.L().Fatal("no crawler host specified")
	}

	contextName := os.Getenv(schemas.EnvVarContextName)

	crawlerDef, exists := crawlers.GetAllDefaults()[crawlerName]
	if !exists || crawlerDef.Crawler == nil {
		zap.L().Fatal(fmt.Sprintf("unable to find crawler with name %s", crawlerName))
	}

	params, err := input.ParseParamsFromEnv(crawlerDef.Definition.Parameters)
	if err != nil {
		zap.L().Fatal("unable to parse parameters from environment", zap.Error(err))
	}

	profile := params[crawlers.ProfileParamName]
	downloaderOverride := params[crawlers.DownloaderParamName]
	sleepDuration := time.Minute * 2

	sleepDurationOverride, ok := params[crawlers.SleepDurationParamName]
	if ok && sleepDurationOverride != "" {
		d, err := time.ParseDuration(sleepDurationOverride)
		if err != nil {
			zap.L().Error("error parsing sleep duration", zap.Error(err))
		} else {
			sleepDuration = d
		}
	}

	l := zap.L().With(
		zap.String("base_url", apiBaseURL),
		zap.String("crawler_name", crawlerName),
	)

	client, err := apiClient.NewClient(apiBaseURL, nil,
		apiClient.TokenFileOpt(os.Getenv(schemas.EnvVarOcularTokenPath), time.Minute*5),
		apiClient.WithContextName(contextName))
	if err != nil {
		l.Fatal("error creating API client", zap.Error(err))
	}

	var (
		queue = make(chan schemas.Target)
		wg    sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		l.Info("starting crawler")
		defer wg.Done()
		defer close(queue)
		if err := crawlerDef.Crawler.Crawl(ctx, params, queue); err != nil {
			zap.L().Error("error running crawler", zap.Error(err))
		}
	}()

	lastRun := time.Now()
	for target := range queue {
		targetL := l.With(
			zap.String("target_identifier", target.Identifier),
			zap.String("target_version", target.Version),
			zap.String("target_downloader", target.Downloader),
			zap.String("target_profile", profile),
		)

		waitRemaining := sleepDuration - time.Since(lastRun)
		if waitRemaining < 0 {
			waitRemaining = 0
		}
		targetL.Debug("sleeping before processing target", zap.Duration("remaining", waitRemaining))
		time.Sleep(waitRemaining)
		if downloaderOverride != "" {
			target.Downloader = downloaderOverride
		}

		targetL.Info("processing target")
		pipeline, err := client.CreatePipeline(ctx, profile, target)
		if err != nil {
			l.Error("error processing target", zap.Error(err))
		} else {
			l.Info("pipeline created", zap.String("pipeline_id", pipeline.ID.String()))
		}
		lastRun = time.Now()
	}

	l.Info("search finished successfully")
}
