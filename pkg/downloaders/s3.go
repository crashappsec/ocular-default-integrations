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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	s3Service "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	AWSConfigFileMountPath = "/ocular/aws/config"
)

type s3 struct{}

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

var _ Downloader = s3{}

func (s3) GetName() string {
	return "s3"
}

func (s3) Download(ctx context.Context, bucketName, version, targetDir string) error {
	l := log.FromContext(ctx)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3Service.NewFromConfig(cfg)

	paginator := s3Service.NewListObjectsV2Paginator(client, &s3Service.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})

	var merr *multierror.Error
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to get page of objects: %w", err)
		}

		for _, obj := range page.Contents {
			err := downloadS3Object(ctx, client, bucketName, *obj.Key, version, targetDir)
			if err != nil {
				l.Error(err, "failed to download object",
					"bucket", bucketName,
					"key", *obj.Key)
				merr = multierror.Append(merr, err)
			}
		}
	}
	return nil
}

func downloadS3Object(
	ctx context.Context,
	client *s3Service.Client,
	bucketName, key, version, localDir string,
) error {
	l := log.FromContext(ctx)
	// Get the object from S3
	input := &s3Service.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}
	if version != "" {
		input.VersionId = aws.String(version)
	}

	output, err := client.GetObject(ctx, input)
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Body.Close(); err != nil {
			l.Error(err, "failed to close response body")
		}
	}()

	localPath := filepath.Clean(filepath.Join(localDir, key))
	if err = os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			l.Error(err, "failed to close response body")
		}
	}()

	// Copy the content to the file
	_, err = io.Copy(file, output.Body)
	return err
}

func (s3) GetMetadataFiles() []string {
	return nil
}
