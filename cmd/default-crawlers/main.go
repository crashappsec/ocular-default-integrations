// Copyright (C) 2025-2026 Crash Override, Inc.
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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	version   = "unknown"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	ctx := context.Background()

	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts)).
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	log.SetLogger(logger)
	ctx = log.IntoContext(ctx, logger)

	logger.Info("starting crawler")

	searchName := os.Getenv(v1beta1.EnvVarSearchName)
	if searchName == "" {
		logger.Error(
			fmt.Errorf("%s environment variable not set", v1beta1.EnvVarSearchName),
			"no search specified",
		)
		os.Exit(1)
	}

	crawlerName := strings.TrimPrefix(os.Getenv(v1beta1.EnvVarCrawlerName), "ocular-defaults-")
	if crawlerName == "" {
		logger.Error(
			fmt.Errorf("%s environment variable not set", v1beta1.EnvVarCrawlerName),
			"no crawler specified",
		)
		os.Exit(1)
	}

	if crawlerOverride := os.Getenv("OCULAR_CRAWLER_NAME_OVERRIDE"); crawlerOverride != "" {
		crawlerName = crawlerOverride
	}

	crawler, found := crawlers.All[crawlerName]
	if !found {
		logger.Error(fmt.Errorf("unknown crawler %s", crawlerName), "no valid crawler specified")
		os.Exit(1)
	}

	params, err := input.ParseParamsFromEnv(crawler.Parameters)
	if err != nil {
		logger.Error(err, "unable to parse parameters from environment")
		os.Exit(1)
	}

	fifo, err := os.OpenFile(os.Getenv(v1beta1.EnvVarPipelineFIFO), syscall.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		logger.Error(err, "unable to open pipeline FIFO")
		os.Exit(1)
	}
	logger = logger.WithValues("crawler", crawler.Name, "params", params)
	logger.Info("executing crawler")

	var queue = make(chan v1beta1.Target)

	go func() {
		defer close(queue)
		if err := crawler.Crawl(ctx, params, queue); err != nil {
			logger.Error(err, "error running crawler")
		}
	}()

	logger.Info("awaiting target discovery")
	encoder := json.NewEncoder(fifo)
	for {
		target, ok := <-queue
		if !ok {
			logger.Info("queue closed, exiting")
			break
		}
		err := encoder.Encode(&target)
		if err != nil {
			logger.Error(err, "unable to encode target JSON", "target", target)
		}
	}

	logger.Info("search finished successfully")
}
