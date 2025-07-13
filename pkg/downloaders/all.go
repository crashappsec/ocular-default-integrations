// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import "github.com/crashappsec/ocular/pkg/schemas"

const (
	GitDownloaderName    = "git"
	DockerDownloaderName = "docker"
	PyPiDownloaderName   = "pypi"
	NpmDownloaderName    = "npm"
	GcsDownloaderName    = "gcs"
	S3DownloaderName     = "s3"
)

var allDownloaders = map[string]DefaultDownloader{
	GitDownloaderName: {
		Downloader: git{},
		Definition: schemas.Downloader{
			Secrets: []schemas.SecretRef{
				{
					Name:        "downloader-gitconfig",
					MountType:   schemas.SecretMountTypeFile,
					MountTarget: "/etc/gitconfig",
				},
			},
		},
	},
	DockerDownloaderName: {
		Downloader: docker{},
		Definition: schemas.Downloader{
			Secrets: []schemas.SecretRef{
				{
					Name:        "downloader-dockerconfig",
					MountType:   schemas.SecretMountTypeFile,
					MountTarget: "/etc/docker/config.json",
				},
			},
			Env: []schemas.EnvVar{
				{
					Name:  "DOCKER_CONFIG",
					Value: "/etc/docker",
				},
			},
		},
	},
	PyPiDownloaderName: {
		Downloader: pypi{},
	},
	NpmDownloaderName: {
		Downloader: npm{},
	},
	GcsDownloaderName: {
		Downloader: gcs{},
		Definition: schemas.Downloader{
			Secrets: []schemas.SecretRef{
				{
					Name:        "downloader-gcs-credentials",
					MountType:   schemas.SecretMountTypeFile,
					MountTarget: "GOOGLE_APPLICATION_CREDENTIALS",
				},
			},
		},
	},
	S3DownloaderName: {
		Downloader: s3{},
		Definition: schemas.Downloader{
			Secrets: []schemas.SecretRef{
				{
					Name:        "downloader-aws-config",
					MountType:   schemas.SecretMountTypeFile,
					MountTarget: "/etc/aws/config",
				},
			},
			Env: []schemas.EnvVar{
				{
					Name:  "AWS_CONFIG_FILE",
					Value: "/etc/aws/config",
				},
			},
		},
	},
}
