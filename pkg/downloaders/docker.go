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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.uber.org/zap"
)

var shaRegex = regexp.MustCompile(`^[a-f0-9]{40}$`)

type docker struct{}

func (docker) Download(ctx context.Context, dockerImage, tag, targetDir string) error {
	var fullImage string
	if shaRegex.MatchString(tag) {
		fullImage = dockerImage + "@" + tag
	} else {
		fullImage = dockerImage + ":" + tag
	}

	ref, err := name.ParseReference(fullImage, name.StrictValidation)
	if err != nil {
		zap.L().Error("Failed to parse reference", zap.String("image", fullImage), zap.Error(err))
		return fmt.Errorf("parsing reference %q: %v", fullImage, err)
	}

	zap.L().Debug("fetching image from remote", zap.String("ref", ref.String()))
	img, err := remote.Image(
		ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	)
	if err != nil {
		zap.L().
			Error("Failed to get remote manifest", zap.String("image", fullImage), zap.Error(err))
		return err
	}

	tar, err := os.Create(filepath.Clean(filepath.Join(targetDir, "target.tar")))
	if err != nil {
		return fmt.Errorf("unable to create tar file: %w", err)
	}

	zap.L().Info("Downloading image", zap.String("image", fullImage))
	if err = tarball.Write(ref, img, tar); err != nil {
		zap.L().Error("Failed to write tarball", zap.String("image", fullImage), zap.Error(err))
		return err
	}
	zap.L().Info("Downloaded image successfully", zap.String("image", fullImage))
	return nil
}
