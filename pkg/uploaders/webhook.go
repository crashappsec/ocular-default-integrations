// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package uploaders

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/pkg/input"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type webhook struct{}

var _ Uploader = webhook{}

func (w webhook) GetName() string {
	return "webhook"
}

func (w webhook) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (w webhook) GetFileSecrets() []definitions.FileSecret {
	return nil
}

func (w webhook) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

const (
	WebhookURLParamName    = "URL"
	WebhookMethodParamName = "METHOD"
)

func (w webhook) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        WebhookURLParamName,
			Description: "URL of the webhook to send data to.",
			Required:    true,
		},
		{
			Name:        WebhookMethodParamName,
			Description: "The HTTP method to use for the webhook request. Defaults to PUT.",
			Required:    false,
			Default:     ptr.To("PUT"),
		},
	}
}

func (w webhook) Upload(
	ctx context.Context,
	_ input.PipelineMetadata,
	params map[string]string,
	files []string,
) error {
	l := log.FromContext(ctx)
	u := params[WebhookURLParamName]
	m := params[WebhookMethodParamName]
	if m == "" {
		m = http.MethodPut
	}

	client := &http.Client{}

	for _, file := range files {
		l.Info(fmt.Sprintf("uploading file %s", file), "method", m, "url", u, "file", file)
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", file, err)
		}
		body, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}
		req, err := http.NewRequestWithContext(ctx, m, u, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				l.Error(err, "failed to close response body")
			}
		}()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to upload file, status code: %d", resp.StatusCode)
		}
	}
	return nil
}
