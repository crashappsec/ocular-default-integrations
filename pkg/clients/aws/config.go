// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RegionParamName  = "AWS_REGION"
	ProfileParamName = "AWS_PROFILE"

	ConfigFileMountPath = "/ocular/aws/config"
)

func GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        RegionParamName,
			Description: "AWS region of the ECR repository. Defaults to the region configured in the AWS SDK.",
			Required:    false,
		},
		{
			Name:        ProfileParamName,
			Description: "ARN of the role to assume for accessing the ECR repository. Optional.",
			Required:    false,
		},
	}
}

func GetAWSFileSecrets() []definitions.FileSecret {
	return []definitions.FileSecret{
		{
			SecretKey: "aws-config",
			MountPath: ConfigFileMountPath,
		},
	}
}

type Options struct {
	Region     string
	AssumeRole string
}

func WithRegionOverride(regionOverride string) func(*config.LoadOptions) error {
	if regionOverride == "" {
		return func(o *config.LoadOptions) error {
			return nil
		}
	}
	return config.WithRegion(regionOverride)
}

func WithProfile(profile string) func(*config.LoadOptions) error {
	if profile != "" {
		return config.WithSharedConfigProfile(profile)
	}

	return func(*config.LoadOptions) error { return nil }
}

func BuildConfig(ctx context.Context, opts ...func(*config.LoadOptions) error) (aws.Config, error) {
	l := log.FromContext(ctx)
	if f, err := os.Stat(ConfigFileMountPath); err == nil && !f.IsDir() {
		opts = append(opts, config.WithSharedConfigFiles([]string{ConfigFileMountPath}))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		l.Error(err, "Failed to load AWS configuration")
		return aws.Config{}, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	return cfg, nil
}
