// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

// Default downloaders bundled with Ocular.
// See the [downloaders] package for more details.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
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

	targetDir := os.Getenv(v1beta1.EnvVarTargetDir)
	targetIdentifier := os.Getenv(v1beta1.EnvVarTargetIdentifier)
	targetVersion := os.Getenv(v1beta1.EnvVarTargetVersion)

	downloaderName := strings.TrimPrefix(os.Getenv(v1beta1.EnvVarDownloaderName), "ocular-defaults-")
	if downloaderName == "" {
		logger.Error(
			fmt.Errorf("%s environment variable not set", v1beta1.EnvVarDownloaderName),
			"no downloader specified",
		)
		os.Exit(1)
	}

	if downloaderOverride := os.Getenv("OCULAR_DOWNLOADER_NAME_OVERRIDE"); downloaderOverride != "" {
		downloaderName = downloaderOverride
	}

	l := logger.WithValues(
		"target_dir", targetDir,
		"downloader", downloaderName,
		"target_identifier", targetIdentifier,
		"target_version", targetVersion,
	)

	logger.Info("starting downloader")

	logger.Info("creating target directory")
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		logger.Error(err, "error creating target directory")
	}

	var downloader downloaders.Downloader
	for _, d := range downloaders.AllDownloaders {
		if d.GetName() == downloaderName {
			downloader = d
			break
		}
	}
	if downloader == nil {
		logger.Error(
			fmt.Errorf("unknown downloader %s", downloaderName),
			"no valid downloader specified",
		)
		os.Exit(1)
	}

	l.Info("downloading target")

	err := downloader.Download(ctx, targetIdentifier, targetVersion, targetDir)
	if err != nil {
		l.Error(err, "error downloading target")
		os.Exit(1)
	}

	l.Info("downloaded target successfully")
}
