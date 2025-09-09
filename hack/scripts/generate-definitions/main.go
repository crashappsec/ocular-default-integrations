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
	"strings"

	"github.com/crashappsec/ocular-default-integrations/pkg/crawlers"
	"github.com/crashappsec/ocular-default-integrations/pkg/downloaders"
	"github.com/crashappsec/ocular-default-integrations/pkg/uploaders"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		"ghcr.io/crashappsec/ocular-default-crawlers",
		"the image to use for the default crawlers",
	)
	flag.StringVar(
		&downloadersImage,
		"downloaders-image",
		"ghcr.io/crashappsec/ocular-default-downloaders",
		"the image to use for the default downloaders",
	)
	flag.StringVar(
		&uploadersImage,
		"uploaders-image",
		"ghcr.io/crashappsec/ocular-default-uploaders",
		"the image to use for the default uploaders",
	)
	flag.StringVar(&imagesTag, "images-tag", "latest", "the tag to use for all images")

	flag.StringVar(&outputFolder, "output-folder", "config", "set the output directory (required)")
}

func main() {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)
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

	downloaderImageStub := "default-downloaders:latest"
	downloaders := downloaders.GenerateObjects(downloaderImageStub)
	if err = createResourceKustomizeFolder[*v1beta1.Downloader]("Downloaders", downloaders); err != nil {
		zap.L().Fatal("error creating downloader kustomize folder", zap.Error(err))
	}

	crawlerImageStub := "default-crawlers:latest"
	crawlers := crawlers.GenerateObjects(crawlerImageStub)
	if err = createResourceKustomizeFolder[*v1beta1.Crawler]("Crawlers", crawlers); err != nil {
		zap.L().Fatal("error creating crawler kustomize folder", zap.Error(err))
	}

	uploaderImageStub := "default-uploaders:latest"
	uploaders := uploaders.GenerateObjects(uploaderImageStub)
	if err = createResourceKustomizeFolder[*v1beta1.Uploader]("Uploaders", uploaders); err != nil {
		zap.L().Fatal("error creating uploader kustomize folder", zap.Error(err))
	}
}

func createResourceKustomizeFolder[T client.Object](kind string, resources []T) error {
	kindLower := strings.ToLower(kind)
	kindFolder := filepath.Join(outputFolder, kindLower)
	if err := os.MkdirAll(kindFolder, 0o750); err != nil {
		zap.L().
			Fatal("resource folder could not be created", zap.Error(err), zap.String("kind", kind))
	}

	var merr *multierror.Error
	for _, resource := range resources {
		filename := fmt.Sprintf("%s.yaml", resource.GetName())
		file, err := os.Create(
			filepath.Clean(filepath.Join(kindFolder, filename)),
		)
		if err != nil {
			zap.L().
				Error("error creating resource", zap.Error(err), zap.String("kind", kind), zap.String("resource", resource.GetName()))
			merr = multierror.Append(merr, err)
		}

		e := json.NewSerializerWithOptions(
			json.DefaultMetaFactory,
			nil,
			nil,
			json.SerializerOptions{Yaml: true, Pretty: true, Strict: false},
		)
		if err = e.Encode(resource, file); err != nil {
			zap.L().
				Error("error printing resource", zap.Error(err), zap.String("kind", kind), zap.String("resource", resource.GetName()))
			merr = multierror.Append(merr, err)
		}

		_ = file.Close()
	}

	if merr != nil {
		return merr.ErrorOrNil()
	}
	return nil
}
