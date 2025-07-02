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

	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/pkg/schemas"
)

type Uploader interface {
	Upload(
		ctx context.Context,
		metadata input.PipelineMetadata,
		params map[string]string,
		files []string,
	) error
}

type DefaultUploader struct {
	Definition schemas.Uploader
	Uploader   Uploader `json:"uploader"`
}
