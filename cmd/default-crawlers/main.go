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
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/crashappsec/ocular-default-integrations/pkg/cli"
	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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

	namespace := os.Getenv(v1beta1.EnvVarNamespaceName)

	var crawler crawlers.Crawler
	for _, c := range crawlers.AllCrawlers {
		if c.GetName() == crawlerName {
			crawler = c
			break
		}
	}
	if crawler == nil {
		logger.Error(fmt.Errorf("unknown crawler %s", crawlerName), "no valid crawler specified")
		os.Exit(1)
	}

	paramDefinitions := input.CombineParameterDefinitions(crawler.GetParameters(), crawlers.DefaultParameters)

	params, err := input.ParseParamsFromEnv(paramDefinitions)
	if err != nil {
		logger.Error(err, "unable to parse parameters from environment")
	}

	profile := params[crawlers.ProfileParamName]
	downloaderOverride := params[crawlers.DownloaderOverrideParamName]
	downloaderKind := params[crawlers.DownloaderOverrideKindParamName]
	scannerServiceAccount := params[crawlers.ScanServiceAccountParamName]
	uploaderServiceAccount := params[crawlers.UploadServiceAccountParamName]

	sleepDuration, err := time.ParseDuration(params[crawlers.SleepDurationParamName])
	if err != nil {
		sleepDuration = time.Minute
		logger.Error(
			err,
			fmt.Sprintf("unable to parse sleep duration, defaulting to %s", sleepDuration.String()),
		)
	}

	ttl, err := time.ParseDuration(params[crawlers.PipelineTTLParamName])
	if err != nil {
		ttl = 24 * time.Hour * 7 // 7 days
		logger.Error(
			err,
			fmt.Sprintf("unable to parse pipeline TTL, defaulting to %s", ttl.String()),
		)
	}

	clientset, err := cli.ParseKubernetesClientset(ctx)
	if err != nil {
		logger.Error(err, "unable to create kubernetes clientset")
		os.Exit(1)
	}

	var (
		queue = make(chan crawlers.CrawledTarget)
		wg    sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		logger.Info("starting crawler", "crawler", crawlerName, "params", params)
		defer wg.Done()
		defer close(queue)
		if err := crawler.Crawl(ctx, params, queue); err != nil {
			logger.Error(err, "error running crawler", "crawler", crawlerName, "params", params)
		}
	}()

	lastRun := time.Now()
	for crawledTarget := range queue {
		target := crawledTarget.Target
		downloader := crawledTarget.DefaultDownloader
		downloader.Name = "ocular-defaults-" + downloader.Name
		if downloaderOverride != "" {
			downloader = corev1.ObjectReference{
				Name: downloaderOverride,
				Kind: downloaderKind,
			}

		}

		targetL := logger.WithValues(
			"target_identifier", target.Identifier,
			"target_version", target.Version,
			"downloader", downloader,
			"profile", profile,
		)

		waitRemaining := sleepDuration - time.Since(lastRun)
		if waitRemaining < 0 {
			waitRemaining = 0
		}
		targetL.Info("sleeping before processing target", "remaining", waitRemaining)
		time.Sleep(waitRemaining)

		targetL.Info("processing target")

		pipeline := &v1beta1.Pipeline{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", searchName),
				Labels: map[string]string{
					"ocular.crashoverride.run/search":  searchName,
					"ocular.crashoverride.run/crawler": crawlerName,
				},
			},
			Spec: v1beta1.PipelineSpec{
				ProfileRef: corev1.ObjectReference{
					Name: profile,
				},
				DownloaderRef:           downloader,
				TTLSecondsAfterFinished: ptr.To[int32](int32(ttl.Seconds())),
				Target:                  target,
			},
		}

		if scannerServiceAccount != "" {
			pipeline.Spec.ScanServiceAccountName = scannerServiceAccount
		}
		if uploaderServiceAccount != "" {
			pipeline.Spec.UploadServiceAccountName = uploaderServiceAccount
		}

		p, err := clientset.ApiV1beta1().
			Pipelines(namespace).
			Create(ctx, pipeline, v1.CreateOptions{})
		if err != nil {
			targetL.Error(err, "error processing target")
		} else {
			targetL.Info("pipeline created", "pipeline_name", p.Name)
		}
		lastRun = time.Now()
	}

	logger.Info("search finished successfully")
}
