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
	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	S3BucketParamName         = "BUCKET"
	S3RegionParamName         = "REGION"
	S3ParentFolderParamName   = "PARENT_FOLDER"
	S3FolderTemplateParamName = "FOLDER_TEMPLATE"

	AWSConfigFileMountPath = "/ocular/aws/config"
)

type s3 struct{}

func (s s3) GetName() string {
	return "s3"
}

func (s s3) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        S3BucketParamName,
			Description: "PipelineName of the S3 bucket to upload to.",
			Required:    true,
		},
		{
			Name:        S3RegionParamName,
			Description: "AWS region of the S3 bucket. Defaults to the region configured in the AWS SDK.",
			Required:    false,
		},
		{
			Name: S3FolderTemplateParamName,
			Description: "Template for the folder structure in the S3 bucket. " +
				"Supports placeholders like {{ .PipelineName }}, {{ .TargetID }}, {{ .TargetVersion }}. " +
				"Using '/' in the template will create nested folders. " +
				"Defaults to '{{ .PipelineName }}'.",
			Required: false,
			Default:  ptr.To("{{ .PipelineName }}"),
		},
	}
}

func (s s3) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (s s3) GetFileSecrets() []definitions.FileSecret {
	return []definitions.FileSecret{
		{
			SecretKey: "aws-config",
			MountPath: AWSConfigFileMountPath,
		},
	}
}

func (s s3) EnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "AWS_CONFIG_FILE",
			Value: AWSConfigFileMountPath,
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
	l := log.FromContext(ctx)
	bucketName := params[S3BucketParamName]
	regionOverride := params[S3RegionParamName]
	folderTemplate, ok := params[S3FolderTemplateParamName]
	if !ok {
		folderTemplate = "{{ .PipelineName }}"
	}

	userTemplater := input.NewUserTemplater("s3 uploader")
	artifactFolder, err := userTemplater.Execute(folderTemplate, metadata)
	if err != nil {
		l.Error(err, "Failed to parse folder template", "template", folderTemplate)
		return fmt.Errorf("failed to parse folder template: %w", err)
	}

	var opts []func(*config.LoadOptions) error
	if f, err := os.Stat(AWSConfigFileMountPath); err == nil && !f.IsDir() {
		opts = append(opts, config.WithSharedConfigFiles([]string{AWSConfigFileMountPath}))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		l.Error(err, "Failed to load AWS configuration")
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
				l.Error(err, "Failed to close file", "file", file)
			}
		}()

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
	}

	if err := merr.ErrorOrNil(); err != nil {
		return fmt.Errorf("failed to upload files to S3: %w", err)
	}

	return nil
}
