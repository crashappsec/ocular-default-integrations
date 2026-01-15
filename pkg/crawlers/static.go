// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package crawlers

import (
	"bufio"
	"context"
	"strings"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var _ Crawler = StaticList{}

type StaticList struct{}

func (StaticList) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (StaticList) GetFileSecrets() []definitions.FileSecret {
	return nil
}
func (StaticList) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

const (
	StaticTargetIdentifierList = "TARGET_IDENTIFIERS"
)

func (s StaticList) Crawl(
	_ context.Context,
	params map[string]string,
	queue chan CrawledTarget,
) error {
	scanner := bufio.NewScanner(strings.NewReader(params[StaticTargetIdentifierList]))
	for scanner.Scan() {
		queue <- CrawledTarget{
			Target: v1beta1.Target{
				Identifier: scanner.Text(),
			},
		}
	}

	return nil
}

func (s StaticList) GetParameters() []v1beta1.ParameterDefinition {
	return []v1beta1.ParameterDefinition{
		{
			Name:        StaticTargetIdentifierList,
			Description: "New line separated list of target identifiers to crawl.",
			Required:    true,
		},
		// This is now set to required, since there is no "default" downloader
		// for arbitrary targets.
		{
			Name:        DownloaderOverrideParamName,
			Description: "Downloader to use for the crawled targets. (required for StaticList crawler)",
			Required:    true,
		},
	}
}

func (s StaticList) GetName() string {
	return "static-list"
}
