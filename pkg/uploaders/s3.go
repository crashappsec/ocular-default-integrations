// Copyright (C) 2025-2026 Crash Override, Inc.
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

	s3Service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/clients/aws"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	S3BucketParamName         = "BUCKET"
	S3FolderTemplateParamName = "FOLDER_TEMPLATE"
)

type s3 struct{}

func (s s3) GetName() string {
	return "s3"
}

func (s s3) GetParameters() []v1beta1.ParameterDefinition {
	return append(aws.GetParameters(),
		v1beta1.ParameterDefinition{
			Name:        S3BucketParamName,
			Description: "PipelineName of the S3 bucket to upload to.",
			Required:    true,
		},
		v1beta1.ParameterDefinition{
			Name: S3FolderTemplateParamName,
			Description: "Template for the folder structure in the S3 bucket. " +
				"Supports placeholders like .PipelineName, .TargetID, .TargetVersion . " +
				"Using '/' in the template will create nested folders. " +
				"Defaults to '.PipelineName' .",
			Required: false,
			Default:  ptr.To(""), // default handled in code, templating gets messed up with helm rendering
		})
}

func (s s3) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (s s3) GetFileSecrets() []definitions.FileSecret {
	return aws.GetAWSFileSecrets()
}

func (s s3) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

var _ Uploader = s3{}

func (s s3) Upload(
	ctx context.Context,
	metadata input.PipelineMetadata,
	params map[string]string,
	files []string,
) error {
	l := log.FromContext(ctx)
	bucketName := params[S3BucketParamName]
	regionOverride := params[aws.RegionParamName]
	profileOverride := params[aws.ProfileParamName]
	folderTemplate, ok := params[S3FolderTemplateParamName]
	if !ok || folderTemplate == "" {
		folderTemplate = "{{ .PipelineName }}"
	}

	userTemplater := input.NewUserTemplater("s3 uploader")
	artifactFolder, err := userTemplater.Execute(folderTemplate, metadata)
	if err != nil {
		l.Error(err, "Failed to parse folder template", "template", folderTemplate)
		return fmt.Errorf("failed to parse folder template: %w", err)
	}

	cfg, err := aws.BuildConfig(ctx, aws.WithProfile(profileOverride), aws.WithRegionOverride(regionOverride))
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	s3Client := s3Service.NewFromConfig(cfg)
	var merr *multierror.Error
	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}

		key := filepath.Join(filepath.Clean(artifactFolder), filepath.Base(file))
		l.Info("putting new object", "bucket", bucketName, "key", key)
		_, err = s3Client.PutObject(ctx, &s3Service.PutObjectInput{
			Bucket: &bucketName,
			Key:    &key,
			Body:   f,
			Metadata: map[string]string{
				"pipelineID":       metadata.PipelineName,
				"downloader":       metadata.DownloaderName,
				"targetIdentifier": metadata.TargetIdentifier,
				"targetVersion":    metadata.TargetVersion,
			},
		})
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("failed to upload file %s: %w", file, err))
		}
		if err = f.Close(); err != nil {
			l.Error(err, "Failed to close file", "file", file)
		}
	}

	if err := merr.ErrorOrNil(); err != nil {
		return fmt.Errorf("failed to upload files to S3: %w", err)
	}

	return nil
}
