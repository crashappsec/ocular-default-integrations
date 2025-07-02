// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package uploaders

import "github.com/crashappsec/ocular/pkg/schemas"

var allUploaders = map[string]DefaultUploader{
	"s3": {
		Definition: schemas.Uploader{
			Parameters: map[string]schemas.ParameterDefinition{
				S3BucketParamName: {
					Description: "Name of the S3 bucket to upload to.",
					Required:    true,
				},
				S3RegionParamName: {
					Description: "AWS region of the S3 bucket. Defaults to the region configured in the AWS SDK.",
					Required:    false,
					Default:     "",
				},
				S3SubFolderParamName: {
					Description: "Subfolder in the S3 bucket to upload files to. Defaults to the root of the bucket.",
					Required:    false,
					Default:     "",
				},
			},
			UserContainer: schemas.UserContainer{
				Secrets: []schemas.SecretRef{
					{
						Name:        "uploader-awsconfig",
						MountTarget: "/root/.aws/config",
						MountType:   schemas.SecretMountTypeFile,
					},
				},
			},
		},
		Uploader: s3{},
	},
	"webhook": {
		Definition: schemas.Uploader{
			Parameters: map[string]schemas.ParameterDefinition{
				WebhookURLParamName: {
					Description: "URL of the webhook to send data to.",
					Required:    true,
				},
				WebhookMethodParamName: {
					Description: "The HTTP method to use for the webhook request. Defaults to PUT.",
					Required:    false,
					Default:     "PUT",
				},
			},
		},
		Uploader: webhook{},
	},
}

func GetAllDefaults() map[string]DefaultUploader {
	result := make(map[string]DefaultUploader, len(allUploaders))
	for name, def := range allUploaders {
		result[name] = def
	}
	return result
}
