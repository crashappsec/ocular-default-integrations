// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PyPIRelease struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

type PyPIPackageInfo struct {
	Version string `json:"version"`
}
type PyPIMetadata struct {
	Releases map[string][]PyPIRelease `json:"releases"`
	Info     PyPIPackageInfo          `json:"info"`
}

type pypi struct{}

func (p pypi) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (p pypi) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (p pypi) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

var _ Downloader = pypi{}

func (pypi) GetName() string {
	return "pypi"
}

func (pypi) Download(ctx context.Context, packageName, version, targetDir string) error {
	l := log.FromContext(ctx)
	// Construct the URL for the PyPI JSON API
	apiURL := fmt.Sprintf("https://pypi.org/pypi/%s/json", packageName)

	// Fetch metadata
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch package metadata: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			l.Error(err, "failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch metadata: %s", resp.Status)
	}

	var metadata PyPIMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if err = os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	if version == "" {
		version = metadata.Info.Version
	}

	files, ok := metadata.Releases[version]
	if !ok {
		return fmt.Errorf("version %s not found for package %s", version, packageName)
	}

	// download each file to the filename from the response, in the target directory
	for _, fileInfo := range files {
		l.Info("downloading file", "filename", fileInfo.Filename, "url", fileInfo.URL)
		if err = downloadFile(ctx, fileInfo.URL, filepath.Join(targetDir, fileInfo.Filename)); err != nil {
			return fmt.Errorf("failed to download file %s: %w", fileInfo.Filename, err)
		}
	}

	manifest, err := os.Create(
		filepath.Clean(filepath.Join(targetDir, fmt.Sprintf("%s-%s.json", packageName, version))),
	)
	if err != nil {
		// just log since this file is not critical
		l.Error(err, "failed to create manifest file")
	} else {
		defer func() {
			if err := manifest.Close(); err != nil {
				l.Error(err, "failed to close response body")
			}
		}()
		if err := json.NewEncoder(manifest).Encode(metadata); err != nil {
			// just log since this file is not critical
			l.Error(err, "failed to write manifest file")
		} else {
			l.Info("wrote manifest file", "path", manifest.Name())
		}
	}

	return nil
}

// downloadFile downloads a file from a URL to a local path
