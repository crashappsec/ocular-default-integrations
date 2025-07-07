// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crashappsec/ocular-default-integrations/internal/config"
	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular-default-integrations/pkg/uploaders"
	"github.com/crashappsec/ocular/pkg/schemas"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	// 	imageTag string is the docker image tag of the images
	imageTag string
	// outputFolder is the folder to write results to.
	// A folder for each of the resource types with defaults (uploader, downloader, crawler)
	// will be created with each default definition being a `.yaml` file with the same name
	// in the respective directory
	outputFolder string
)

func init() {
	flag.StringVar(&imageTag, "image-tag", "latest", "override the tag of the docker images")
	flag.StringVar(&outputFolder, "output-folder", "", "set the output directory (required)")
	config.InitLogger(os.Getenv("OCULAR_LOGGING_LEVEL"), os.Getenv("OCULAR_LOGGING_FORMAT"))
}

func main() {
	zap.L().Info("generating definitions")
	flag.Parse()

	if outputFolder == "" {
		zap.L().Fatal("--output-folder is required")
	}

	f, err := os.Stat(outputFolder)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		zap.L().Fatal("--output-folder returned an error when checking for validity")
	}
	if err == nil && !f.IsDir() {
		zap.L().Fatal("--output-folder already exists is not a directory")
	}

	if err = os.MkdirAll(outputFolder, 0o750); err != nil {
		zap.L().Fatal("--output-folder could not be created", zap.Error(err))
	}

	downloaders := generateDownloaderDefinitions(imageTag)
	downloaderFolder := filepath.Join(outputFolder, "downloaders")
	if err = os.MkdirAll(downloaderFolder, 0o750); err != nil {
		zap.L().Fatal("downloader folder could not be created", zap.Error(err))
	}
	for name, downloader := range downloaders {
		file, err := os.Create(
			filepath.Clean(filepath.Join(downloaderFolder, fmt.Sprintf("%s.yaml", name))),
		)
		if err != nil {
			zap.L().
				Error("error creating downloader", zap.Error(err), zap.String("downloader", name))
			continue
		}

		if err = yaml.NewEncoder(file).Encode(downloader); err != nil {
			zap.L().
				Error("error encoding downloader", zap.Error(err), zap.String("downloader", name))
		}

		_ = file.Close()
	}

	crawlers := generateCrawlerDefinitions(imageTag)
	crawlerFolder := filepath.Join(outputFolder, "crawlers")
	if err = os.MkdirAll(crawlerFolder, 0o750); err != nil {
		zap.L().Fatal("crawler folder could not be created", zap.Error(err))
	}
	for name, crawler := range crawlers {
		file, err := os.Create(
			filepath.Clean(filepath.Join(crawlerFolder, fmt.Sprintf("%s.yaml", name))),
		)
		if err != nil {
			zap.L().Error("error creating crawler", zap.Error(err), zap.String("crawler", name))
			continue
		}

		if err = yaml.NewEncoder(file).Encode(crawler); err != nil {
			zap.L().Error("error encoding crawler", zap.Error(err), zap.String("crawler", name))
		}

		_ = file.Close()
	}

	uploaders := generateUploaderDefinitions(imageTag)
	uploaderFolder := filepath.Join(outputFolder, "uploaders")
	if err = os.MkdirAll(uploaderFolder, 0o750); err != nil {
		zap.L().Fatal("uploader folder could not be created", zap.Error(err))
	}
	for name, uploader := range uploaders {
		file, err := os.Create(
			filepath.Clean(filepath.Join(uploaderFolder, fmt.Sprintf("%s.yaml", name))),
		)
		if err != nil {
			zap.L().Error("error creating uploader", zap.Error(err), zap.String("uploader", name))
			continue
		}

		if err = yaml.NewEncoder(file).Encode(uploader); err != nil {
			zap.L().Error("error encoding uploader", zap.Error(err), zap.String("uploader", name))
		}

		_ = file.Close()
	}
}

func generateDownloaderDefinitions(imageTag string) map[string]schemas.Downloader {
	result := make(map[string]schemas.Downloader)
	for downloader, spec := range downloaders.GetAllDefaults() {
		def := spec.Definition
		def.Image = "ghcr.io/crashappsec/ocular-default-downloaders:" + imageTag
		result[downloader] = def
	}
	return result
}

func generateCrawlerDefinitions(imageTag string) map[string]schemas.Crawler {
	result := make(map[string]schemas.Crawler)
	for downloader, spec := range crawlers.GetAllDefaults() {
		def := spec.Definition
		def.Image = "ghcr.io/crashappsec/ocular-default-crawlers:" + imageTag
		result[downloader] = def
	}
	return result
}

func generateUploaderDefinitions(imageTag string) map[string]schemas.Uploader {
	result := make(map[string]schemas.Uploader)
	for downloader, spec := range uploaders.GetAllDefaults() {
		def := spec.Definition
		def.Image = "ghcr.io/crashappsec/ocular-default-uploaders:" + imageTag
		result[downloader] = def
	}
	return result
}
