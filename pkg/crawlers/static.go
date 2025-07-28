// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package crawlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/crashappsec/ocular/pkg/schemas"
)

var _ Crawler = StaticList{}

type StaticList struct{}

/**************
 * Parameters *
 **************/

const (
	StaticTargetIdentifierList = "TARGET_IDENTIFIERS"
)

func (s StaticList) Crawl(
	_ context.Context,
	params map[string]string,
	queue chan schemas.Target,
) error {
	var targetIdentifiers []string
	err := json.Unmarshal([]byte(params[StaticTargetIdentifierList]), &targetIdentifiers)
	if err != nil {
		return fmt.Errorf("failed to unmarshal target identifiers: %w", err)
	}

	for _, targetIdentifier := range targetIdentifiers {
		queue <- schemas.Target{
			Identifier: targetIdentifier,
		}
	}

	return nil
}
