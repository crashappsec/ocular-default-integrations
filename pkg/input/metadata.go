// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package input

import (
	"fmt"
	"os"

	"github.com/crashappsec/ocular/api/v1beta1"
)

type PipelineMetadata struct {
	PipelineName     string `json:"pipelineName"`
	TargetIdentifier string `json:"targetIdentifier"`
	TargetVersion    string `json:"targetVersion,omitempty"`
	DownloaderName   string `json:"downloaderName"`
}

func ParseMetadataFromEnv() (PipelineMetadata, error) {
	metadata := PipelineMetadata{
		PipelineName:     os.Getenv(v1beta1.EnvVarPipelineName),
		TargetIdentifier: os.Getenv(v1beta1.EnvVarTargetIdentifier),
		TargetVersion:    os.Getenv(v1beta1.EnvVarTargetVersion),
		DownloaderName:   os.Getenv(v1beta1.EnvVarDownloaderName),
	}

	if metadata.PipelineName == "" {
		return metadata, fmt.Errorf(
			"missing required environment variable %s",
			v1beta1.EnvVarPipelineName,
		)
	}
	if metadata.TargetIdentifier == "" {
		return metadata, fmt.Errorf(
			"missing required environment variable %s",
			v1beta1.EnvVarTargetIdentifier,
		)
	}

	if metadata.DownloaderName == "" {
		return metadata, fmt.Errorf(
			"missing required environment variable %s",
			v1beta1.EnvVarDownloaderName,
		)
	}

	return metadata, nil
}
