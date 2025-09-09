// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package uploaders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	s3Service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

/**************
 * Parameters *
 **************/

const (
	S3BucketParamName    = "BUCKET"
	S3RegionParamName    = "REGION"
	S3SubFolderParamName = "SUBFOLDER"

	AWSConfigFileMountPath = "/aws/config"
)

type s3 struct{}

func (s s3) GetName() string {
	return "s3"
}

func (s s3) GetParameters() map[string]v1beta1.ParameterDefinition {
	return map[string]v1beta1.ParameterDefinition{
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
	}
}

var _ Uploader = s3{}

func (s s3) Upload(
	ctx context.Context,
	metadata input.PipelineMetadata,
	params map[string]string,
	files []string,
) error {
	bucketName := params[S3BucketParamName]
	regionOverride := params[S3RegionParamName]
	subFolder := params[S3SubFolderParamName]

	var opts []func(*config.LoadOptions) error
	if f, err := os.Stat(AWSConfigFileMountPath); err == nil && !f.IsDir() {
		opts = append(opts, config.WithSharedConfigFiles([]string{AWSConfigFileMountPath}))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		zap.L().Error("Failed to load AWS configuration", zap.Error(err))
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if regionOverride != "" {
		cfg.Region = regionOverride
	}

	s3Client := s3Service.NewFromConfig(cfg)
	var merr *multierror.Error
	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				zap.L().Error("Failed to close file", zap.String("file", file), zap.Error(err))
			}
		}()

		key := filepath.Join(subFolder, metadata.ID, filepath.Base(file))

		_, err = s3Client.PutObject(ctx, &s3Service.PutObjectInput{
			Bucket: &bucketName,
			Key:    &key,
			Body:   f,
			Metadata: map[string]string{
				"pipelineID":       metadata.ID,
				"targetIdentifier": metadata.TargetIdentifier,
				"targetVersion":    metadata.TargetVersion,
			},
		})
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("failed to upload file %s: %w", file, err))
		}
	}

	if err := merr.ErrorOrNil(); err != nil {
		return fmt.Errorf("failed to upload files to S3: %w", err)
	}

	return nil
}
