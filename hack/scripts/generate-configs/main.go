// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular-default-integrations/pkg/uploaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	// 	crawlerImage is the docker image tag of the default crawlers
	crawlersImage string
	// uploaderImage is the docker image tag of the default uploaders
	uploadersImage string
	// downloaderImage is the docker image tag of the default downloaders
	downloadersImage string

	// imagesTag is the tag to use for all images
	imagesTag string

	// outputFolder is the folder to write results to.
	// A folder for each of the resource types with defaults (uploader, downloader, crawler)
	// will be created with each default definition being a `.yaml` file with the same name
	// in the respective directory
	outputFolder string
)

func init() {
	flag.StringVar(
		&crawlersImage,
		"crawlers-image",
		"crawlers",
		"the image to use for the default crawlers",
	)
	flag.StringVar(
		&downloadersImage,
		"downloaders-image",
		"downloaders",
		"the image to use for the default downloaders",
	)
	flag.StringVar(
		&uploadersImage,
		"uploaders-image",
		"uploaders",
		"the image to use for the default uploaders",
	)
	flag.StringVar(&imagesTag, "images-tag", "latest", "the tag to use for all images")

	flag.StringVar(&outputFolder, "output-folder", "config", "set the output directory (required)")
}

func main() {
	ctx := context.Background()

	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&zap.Options{}))
	log.SetLogger(logger)
	ctx = log.IntoContext(ctx, logger)
	logger.Info("generating definitions")
	flag.Parse()

	if outputFolder == "" {
		logger.Error(fmt.Errorf("--output-folder is required"), "no output folder specified")
		os.Exit(1)
	}

	f, err := os.Stat(outputFolder)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Error(err, "--output-folder returned an error when checking for validity")
		os.Exit(1)
	}
	if err == nil && !f.IsDir() {
		err := fmt.Errorf("--output-folder already exists is not a directory")
		logger.Error(err, "unable to continue with output directory")
		os.Exit(1)
	}

	if err = os.MkdirAll(outputFolder, 0o750); err != nil {
		logger.Error(err, "--output-folder could not be created")
		os.Exit(1)
	}

	downloaderObjs := downloaders.GenerateObjects(downloadersImage, "downloader-secrets")
	if err = createResourceKustomizeFolder[*v1beta1.Downloader](ctx, "Downloaders", downloaderObjs); err != nil {
		logger.Error(err, "error creating downloader kustomize folder")
		os.Exit(1)
	}

	crawlerObjs := crawlers.GenerateObjects(crawlersImage, "crawler-secrets")
	if err = createResourceKustomizeFolder[*v1beta1.Crawler](ctx, "Crawlers", crawlerObjs); err != nil {
		logger.Error(err, "error creating crawler kustomize folder")
		os.Exit(1)
	}

	uploaderObjs := uploaders.GenerateObjects(uploadersImage, "uploader-secrets")
	if err = createResourceKustomizeFolder[*v1beta1.Uploader](ctx, "Uploaders", uploaderObjs); err != nil {
		logger.Error(err, "error creating uploader kustomize folder")
		os.Exit(1)
	}
}

func createResourceKustomizeFolder[T client.Object](ctx context.Context, kind string, resources []T) error {
	l := log.FromContext(ctx).WithValues("kind", kind, "count", len(resources))
	kindLower := strings.ToLower(kind)
	kindFolder := filepath.Join(outputFolder, kindLower)
	if err := os.MkdirAll(kindFolder, 0o750); err != nil {
		return fmt.Errorf("error creating %s folder: %w", kindLower, err)
	}

	var merr *multierror.Error
	for _, resource := range resources {
		filename := fmt.Sprintf("%s.yaml", resource.GetName())
		file, err := os.Create(
			filepath.Clean(filepath.Join(kindFolder, filename)),
		)
		if err != nil {
			l.Error(err, "error creating resource file", "resource", resource.GetName())
			merr = multierror.Append(merr, err)
		}

		e := json.NewSerializerWithOptions(
			json.DefaultMetaFactory,
			nil,
			nil,
			json.SerializerOptions{Yaml: true, Pretty: true, Strict: false},
		)
		if err = e.Encode(resource, file); err != nil {
			l.Error(err, "error printing resource", "resource", resource.GetName())
			merr = multierror.Append(merr, err)
		}

		_ = file.Close()
	}

	if merr != nil {
		return merr.ErrorOrNil()
	}
	return nil
}
