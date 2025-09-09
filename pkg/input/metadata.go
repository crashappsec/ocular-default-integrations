// Copyright (C) 2025 Crash Override, Inc.
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
	ID               string
	TargetIdentifier string
	TargetVersion    string
}

func ParseMetadataFromEnv() (PipelineMetadata, error) {
	metadata := PipelineMetadata{
		ID:               os.Getenv(v1beta1.EnvVarPipelineName),
		TargetIdentifier: os.Getenv(v1beta1.EnvVarTargetIdentifier),
		TargetVersion:    os.Getenv(v1beta1.EnvVarTargetVersion),
	}

	if metadata.ID == "" {
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

	return metadata, nil
}
