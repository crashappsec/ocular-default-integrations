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
	"os"

	"github.com/crashappsec/ocular-default-integrations/internal/config"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular-default-integrations/pkg/uploaders"
	"github.com/crashappsec/ocular/pkg/schemas"
	"go.uber.org/zap"
)

func init() {
	config.Init()
}

func main() {
	resultsDir := os.Getenv(schemas.EnvVarResultsDir)
	uploaderName := os.Getenv(schemas.EnvVarUploaderName)
	metadata, err := input.ParseMetadataFromEnv()
	if err != nil {
		zap.L().Fatal("failed to parse metadata from environment", zap.Error(err))
	}

	var files []string
	for i, arg := range os.Args {
		// files are passed as positional arguments after '--'
		if arg == "--" {
			files = os.Args[i+1:]
			break
		}
	}

	l := zap.L().With(
		zap.String("results_dir", resultsDir),
		zap.String("uploader", uploaderName),
		zap.Strings("files", files),
	)

	l.Info("starting uploader")

	ctx := context.Background()

	uploaderDef, exists := uploaders.GetAllDefaults()[uploaderName]
	if !exists {
		l.Fatal("unknown uploader", zap.String("uploader", uploaderName))
	}

	params, err := input.ParseParamsFromEnv(uploaderDef.Definition.Parameters)
	if err != nil {
		l.Fatal("failed to parse parameters from environment", zap.Error(err))
	}

	var validatedFiles []string
	for _, file := range files {
		l.Debug("validating file exists", zap.String("file", file))
		if _, err := os.Stat(file); os.IsNotExist(err) {
			l.Warn("file does not exist, skipping", zap.String("file", file))
		} else if err != nil {
			validatedFiles = append(validatedFiles, file)
		} else {
			l.Warn("unable to stat file, skipping", zap.String("file", file), zap.Error(err))
		}
	}

	uploader := uploaderDef.Uploader
	err = uploader.Upload(ctx, metadata, params, validatedFiles)
	if err != nil {
		l.Fatal("failed to upload files", zap.Strings("files", files), zap.Error(err))
	}

	l.Info("uploaded artifacts successfully")
}
