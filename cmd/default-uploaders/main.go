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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular-default-integrations/pkg/uploaders"
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

	logger := zap.New(zap.UseFlagOptions(&zap.Options{})).
		WithValues("version", version, "buildTime", buildTime, "gitCommit", gitCommit)
	log.SetLogger(logger)
	ctx = log.IntoContext(ctx, logger)

	logger.Info("starting uploader")

	var files []string
	for i, arg := range os.Args {
		// files are passed as positional arguments after '--'
		if arg == "--" {
			files = os.Args[i+1:]
			break
		}
	}
	resultsDir := os.Getenv(v1beta1.EnvVarResultsDir)

	uploaderName := strings.TrimPrefix(os.Getenv(v1beta1.EnvVarUploaderName), "ocular-defaults-")
	if uploaderName == "" {
		logger.Error(
			fmt.Errorf("%s environment variable not set", v1beta1.EnvVarUploaderName),
			"no uploader specified",
		)
		os.Exit(1)
	}

	l := logger.WithValues(
		"results_dir", resultsDir,
		"uploader", uploaderName,
		"file-args", files,
	)

	metadata, err := input.ParseMetadataFromEnv()
	if err != nil {
		logger.Error(err, "failed to parse metadata from environment")
		os.Exit(1)
	}

	var uploader uploaders.Uploader
	for _, u := range uploaders.AllUploaders {
		if u.GetName() == uploaderName {
			uploader = u
			break
		}
	}

	if uploader == nil {
		logger.Error(fmt.Errorf("unknown uploader %s", uploaderName), "no valid uploader specified")
		os.Exit(1)
	}

	logger.WithValues("uploader", uploaderName).Info("begin upload process")

	params, err := input.ParseParamsFromEnv(uploader.GetParameters())
	if err != nil {
		logger.Error(err, "unable to parse parameters from environment")
	}

	var validatedFiles []string
	for _, file := range files {
		l.Info("validating file exists", "file", file)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			logger.Info("file does not exist, skipping", "file", file)
		} else if err != nil {
			logger.Error(err, "unable to stat file, skipping", "file", file)
		} else {
			validatedFiles = append(validatedFiles, file)
		}
	}

	l.Info("uploading files", "files", validatedFiles)
	err = uploader.Upload(ctx, metadata, params, validatedFiles)
	if err != nil {
		l.Error(err, "failed to upload files", "files", validatedFiles)
		os.Exit(1)
	}

	l.Info("uploaded artifacts successfully")
}
