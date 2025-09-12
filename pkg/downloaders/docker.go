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
	"os"
	"path/filepath"
	"regexp"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var shaRegex = regexp.MustCompile(`^[a-f0-9]{40}$`)

type docker struct{}

var _ Downloader = docker{}

func (docker) GetName() string {
	return "docker"
}

const (
	DockerConfigFolder = "/ocular/docker"
)

func (docker) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (docker) GetFileSecrets() []definitions.FileSecret {
	return []definitions.FileSecret{
		{
			SecretKey: "dockerconfig",
			MountPath: DockerConfigFolder + "/config.json",
		},
	}
}

func (docker) EnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "DOCKER_CONFIG",
			Value: DockerConfigFolder,
		},
	}
}

func (docker) Download(ctx context.Context, dockerImage, tag, targetDir string) error {
	l := log.FromContext(ctx)
	var fullImage string
	if shaRegex.MatchString(tag) {
		fullImage = dockerImage + "@" + tag
	} else {
		fullImage = dockerImage + ":" + tag
	}

	ref, err := name.ParseReference(fullImage, name.StrictValidation)
	if err != nil {
		l.Error(err, "Failed to parse reference", "image", fullImage)
		return fmt.Errorf("parsing reference %q: %v", fullImage, err)
	}

	l.Info("fetching image from remote", "ref", ref.String())
	img, err := remote.Image(
		ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	)
	if err != nil {
		l.Error(err, "Failed to get remote manifest", "image", fullImage)
		return err
	}

	tar, err := os.Create(filepath.Clean(filepath.Join(targetDir, "target.tar")))
	if err != nil {
		return fmt.Errorf("unable to create tar file: %w", err)
	}

	l.Info("Downloading image", "image", fullImage)
	if err = tarball.Write(ref, img, tar); err != nil {
		l.Error(err, "Failed to write tarball", "image", fullImage)
		return err
	}
	l.Info("Downloaded image successfully", "image", fullImage)
	return nil
}
