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
	"net/http"
	"os"
	"path/filepath"

	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	"go.uber.org/zap"
)

type webhook struct{}

var _ Uploader = webhook{}

func (w webhook) GetName() string {
	return "webhook"
}

/**************
 * Parameters *
 **************/

const (
	WebhookURLParamName    = "URL"
	WebhookMethodParamName = "METHOD"
)

func (w webhook) GetParameters() map[string]v1beta1.ParameterDefinition {
	return map[string]v1beta1.ParameterDefinition{
		WebhookURLParamName: {
			Description: "URL of the webhook to send data to.",
			Required:    true,
		},
		WebhookMethodParamName: {
			Description: "The HTTP method to use for the webhook request. Defaults to PUT.",
			Required:    false,
			Default:     "PUT",
		},
	}
}

func (w webhook) Upload(
	ctx context.Context,
	_ input.PipelineMetadata,
	params map[string]string,
	files []string,
) error {
	u := params[WebhookURLParamName]
	m := params[WebhookMethodParamName]
	if m == "" {
		m = http.MethodPut
	}

	client := &http.Client{}

	for _, file := range files {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		req, err := http.NewRequestWithContext(ctx, m, u, f)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				zap.L().Error("failed to close response body", zap.Error(err))
			}
		}()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to upload file, status code: %d", resp.StatusCode)
		}
	}
	return nil
}
