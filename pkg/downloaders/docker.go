// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular/api/v1beta1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var shaRegex = regexp.MustCompile(`^sha256:[a-f0-9]{40}$`)

type Docker struct{}

var _ Downloader = Docker{}

func (Docker) GetName() string {
	return "docker"
}

const (
	DockerConfigFolder = "/ocular/docker"
)

func (Docker) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (Docker) GetFileSecrets() []definitions.FileSecret {
	return []definitions.FileSecret{
		{
			SecretKey: "dockerconfig",
			MountPath: DockerConfigFolder + "/config.json",
		},
	}
}

func (Docker) EnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "DOCKER_CONFIG",
			Value: DockerConfigFolder,
		},
	}
}

func (Docker) Download(ctx context.Context, dockerImage, tag, targetDir string) error {
	if tag == "" {
		tag = "latest"
	}
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

	tarFile, err := os.Create(filepath.Clean(filepath.Join(targetDir, "target.tar")))
	if err != nil {
		return fmt.Errorf("unable to create tar file: %w", err)
	}
	defer func() {
		if err := tarFile.Close(); err != nil {
			l.Error(err, "failed to close tar reader")
		}
	}()

	l.Info("downloading image", "image", fullImage)
	if err = tarball.Write(ref, img, tarFile); err != nil {
		l.Error(err, "Failed to write tarball", "image", fullImage)
		return err
	}
	l.Info("Downloaded image successfully", "image", fullImage)

	metadata := DockerMetadata{
		Image: dockerImage,
		Tag:   tag,
	}

	sha, err := img.Digest()
	if err != nil {
		l.Error(err, "Failed to get image digest", "image", fullImage)
	} else {
		metadata.SHA = sha.String()
	}

	if err = writeJSONStruct(DockerMetadataPath, metadata); err != nil {
		l.Error(err, "Failed to write docker metadata", "path", DockerMetadataPath)
	}

	l.Info("beginning chalk extraction", "image", fullImage)

	layers, err := img.Layers()
	if err != nil {
		l.Error(err, "failed to inspect image layers", "image", fullImage)
		return nil
	}

	if len(layers) == 0 {
		l.Info("image has no layers", "image", fullImage)
		return nil
	}

	if lastLayer := layers[len(layers)-1]; lastLayer != nil {
		l.Info("attempting to extract chalk metadata from last layer", "image", fullImage)
		mediaType, err := lastLayer.MediaType()
		if err != nil {
			l.Error(err, "failed to get last layer media type", "image", fullImage)
			return nil
		}

		if !mediaType.IsLayer() {
			l.Info("last layer media type is not a layer, skipping chalk extraction", "mediaType", mediaType)
			return nil
		}

		rc, err := lastLayer.Uncompressed()
		if err != nil {
			l.Error(err, "failed to get uncompressed last layer", "image", fullImage)
			return nil
		}

		defer func() {
			if err = rc.Close(); err != nil {
				l.Error(err, "failed to close last layer reader")
			}
		}()

		if err = extractChalk(ctx, rc, DockerChalkMetadataPath); err != nil {
			l.Error(err, "failed to extract chalk metadata", "image", fullImage)
		}
	}

	return nil
}

func extractChalk(ctx context.Context, tr io.Reader, chalkPath string) error {
	l := log.FromContext(ctx)
	tarReader := tar.NewReader(tr)
	chalk, err := tarReader.Next()
	if err != nil {
		return fmt.Errorf("reading last layer tar: %w", err)
	}

	if chalk.Name != chalkFileName {
		l.Info("chalk metadata file not found in last layer", "expected", chalkFileName, "found", chalk.Name)
		return nil
	}

	chalkFile, err := os.Create(filepath.Clean(chalkPath))
	if err != nil {
		return fmt.Errorf("creating chalk metadata file: %w", err)
	}
	defer func() {
		if err := chalkFile.Close(); err != nil {
			l.Error(err, "failed to close chalk metadata file")
		}
	}()

	_, err = io.Copy(chalkFile, tarReader)
	if err != nil {
		return fmt.Errorf("writing chalk metadata file: %w", err)
	}
	return nil
}

type DockerMetadata struct {
	Image string `json:"image,omitempty"`
	Tag   string `json:"tag,omitempty"`
	SHA   string `json:"sha,omitempty"`
}

const (
	chalkFileName           = "chalk.json"
	DockerMetadataPath      = v1beta1.PipelineMetadataDirectory + "/docker.json"
	DockerChalkMetadataPath = v1beta1.PipelineMetadataDirectory + "/" + chalkFileName
)

func (Docker) GetMetadataFiles() []string {
	return []string{DockerMetadataPath, DockerChalkMetadataPath}
}
