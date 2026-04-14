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

	"github.com/crashappsec/ocular/api/v1beta1"
)

func init() {
	All.registerCrawler(StaticList)
}

var StaticList = Crawler{
	Name: "static-list",
	Parameters: []v1beta1.ParameterDefinition{
		{
			Name:        StaticTargetIdentifierList,
			Description: "New line separated list of target identifiers to crawl.",
		},
	},
	Crawl: crawlStaticList,
}

const (
	StaticTargetIdentifierList = "TARGET_IDENTIFIERS"
)

func crawlStaticList(
	_ context.Context,
	params map[string]string,
	queue chan v1beta1.Target,
) error {
	scanner := bufio.NewScanner(strings.NewReader(params[StaticTargetIdentifierList]))
	for scanner.Scan() {
		queue <- v1beta1.Target{
			Identifier: scanner.Text(),
		}
	}

	return nil
}
