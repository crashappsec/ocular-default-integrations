// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package input

import (
	"fmt"
	"os"

	"github.com/crashappsec/ocular/api/v1beta1"
	ocularRuntime "github.com/crashappsec/ocular/pkg/runtime"
	"github.com/hashicorp/go-multierror"
)

func ParseParamsFromEnv(
	definitions []v1beta1.ParameterDefinition,
) (map[string]string, error) {
	params := make(map[string]string)

	var merr *multierror.Error
	for _, def := range definitions {
		envValue, exists := os.LookupEnv(ocularRuntime.ParameterToEnvironmentVariable(def.Name))
		if def.Required && !exists {
			merr = multierror.Append(merr, fmt.Errorf("parameter %s is required", def.Name))
			continue
		}
		var value string
		if def.Default != nil {
			value = *def.Default
		}
		if exists {
			value = envValue
		}
		params[def.Name] = value
	}

	return params, merr.ErrorOrNil()
}

func CombineParameterDefinitions(
	lists ...[]v1beta1.ParameterDefinition,
) []v1beta1.ParameterDefinition {
	seen := make(map[string]struct{})
	var combined []v1beta1.ParameterDefinition
	for _, list := range lists {
		for _, def := range list {
			if _, exists := seen[def.Name]; exists {
				continue
			}
			seen[def.Name] = struct{}{}
			combined = append(combined, def)
		}
	}
	return combined
}
