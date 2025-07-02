// Copyright (C) 2025 Crash Override, Inc.
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
	"fmt"
	"os"

	"github.com/crashappsec/ocular-default-integrations/internal/config"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular/pkg/schemas"
	"go.uber.org/zap"
)

func init() {
	config.Init()
}

func main() {
	targetDir := os.Getenv(schemas.EnvVarTargetDir)
	targetDownloader := os.Getenv(schemas.EnvVarTargetDownloader)
	targetIdentifier := os.Getenv(schemas.EnvVarTargetIdentifier)
	targetVersion := os.Getenv(schemas.EnvVarTargetVersion)

	l := zap.L().With(
		zap.String("target_dir", targetDir),
		zap.String("target_downloader", targetDownloader),
		zap.String("target_identifier", targetIdentifier),
		zap.String("target_version", targetVersion),
	)

	l.Info("starting downloader")

	ctx := context.Background()

	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		l.Fatal("error creating target directory", zap.Error(err))
	}

	downloaderDef, exists := downloaders.GetAllDefaults()[targetDownloader]
	if !exists {
		zap.L().Fatal(fmt.Sprintf("unable to find downloader with name %s", targetDownloader))
	}

	err := downloaderDef.Downloader.Download(ctx, targetIdentifier, targetVersion, targetDir)
	if err != nil {
		l.Fatal("error downloading targets", zap.Error(err))
	}

	l.Info("downloaded target successfully")
}
