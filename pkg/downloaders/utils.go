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
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func urlToReader(url string) (io.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL: %s", resp.Status)
	}
	readWriter := bytes.NewBuffer(nil)
	if _, err = io.Copy(readWriter, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return readWriter, nil
}

func downloadFile(url, fpath string) error {
	urlReader, err := urlToReader(url)
	if err != nil {
		return err
	}

	out, err := os.Create(filepath.Clean(fpath))
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			zap.L().Error("failed to close response body", zap.Error(err))
		}
	}()

	_, err = io.Copy(out, urlReader)
	return err
}

func writeTar(tr *tar.Reader, parentDir string) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		targetPath := filepath.Clean(filepath.Join(parentDir, filepath.Clean(header.Name)))

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(targetPath, os.FileMode(header.Mode%math.MaxUint32)); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
				return fmt.Errorf("creating parent directory: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("creating file: %w", err)
			}
			_, err = io.Copy(outFile, tr)
			defer func() {
				if err := outFile.Close(); err != nil {
					zap.L().Error("failed to close response body", zap.Error(err))
				}
			}()
			if err != nil {
				return fmt.Errorf("writing file: %w", err)
			}

			// Set file permissions
			if err = os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("setting permissions: %w", err)
			}
		default:
			zap.L().
				Info("skipping unsupported file type", zap.String("typeflag", string(header.Typeflag)))
		}
	}
	return nil
}
