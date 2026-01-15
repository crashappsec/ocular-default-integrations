// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NpmMetadata struct {
	Versions map[string]struct {
		Dist struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	} `json:"versions"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
}

type npm struct{}

var _ Downloader = npm{}

func (npm) GetName() string {
	return "npm"
}
func (npm) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (npm) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (npm) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

func (npm) Download(ctx context.Context, packageName, version, targetDir string) error {
	l := log.FromContext(ctx)
	registryURL := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)

	resp, err := http.Get(registryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch package metadata: %w", err)
	}
	func() {
		if err := resp.Body.Close(); err != nil {
			l.Error(err, "failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch metadata: %s", resp.Status)
	}

	var metadata NpmMetadata
	if err = json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if version == "" {
		version = metadata.DistTags.Latest
	}

	versionData, ok := metadata.Versions[version]
	if !ok {
		return fmt.Errorf("version %s not found for package %s", version, packageName)
	}

	var tarballReader io.Reader
	tarballReader, err = urlToReader(ctx, versionData.Dist.Tarball)
	if err != nil {
		return fmt.Errorf("failed to fetch tarball: %w", err)
	}

	if strings.HasSuffix(versionData.Dist.Tarball, ".tgz") ||
		strings.HasSuffix(versionData.Dist.Tarball, ".tar.gz") {
		tarballReader, err = gzip.NewReader(tarballReader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
	}

	tr := tar.NewReader(tarballReader)
	if err = writeTar(ctx, tr, targetDir); err != nil {
		return fmt.Errorf("failed to write tarball as fs: %w", err)
	}

	return nil
}

func (npm) GetMetadataFiles() []string {
	return nil
}
