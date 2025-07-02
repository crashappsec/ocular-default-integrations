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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

type gcs struct{}

func (gcs) Download(ctx context.Context, bucketName, _, targetDir string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			zap.L().Error("unable to close client", zap.Error(err))
		}
	}()

	bucket := client.Bucket(bucketName)
	query := &storage.Query{}
	it := bucket.Objects(ctx, query)

	for {
		objAttrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing objects: %w", err)
		}

		localPath := filepath.Join(targetDir, objAttrs.Name)
		if err = os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
			return fmt.Errorf("failed to create local directory: %w", err)
		}

		zap.L().
			Debug("downloading file to local", zap.String("file", objAttrs.Name), zap.String("localPath", localPath))
		if err = downloadGCSObject(ctx, bucket, objAttrs.Name, localPath); err != nil {
			return fmt.Errorf("failed to download object %s: %w", objAttrs.Name, err)
		}
	}

	return nil
}

// downloadGCSObject downloads a single object from GCS and saves it to the given local path.
func downloadGCSObject(
	ctx context.Context,
	bucket *storage.BucketHandle,
	objectName, destPath string,
) error {
	rc, err := bucket.Object(objectName).NewReader(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()

	f, err := os.Create(filepath.Clean(destPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()

	_, err = io.Copy(f, rc)
	return err
}
