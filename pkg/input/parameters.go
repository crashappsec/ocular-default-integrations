// Copyright (C) 2025 Crash Override, Inc.
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
	"github.com/hashicorp/go-multierror"
)

func ParseParamsFromEnv(
	definitions map[string]v1beta1.ParameterDefinition,
) (map[string]string, error) {
	params := make(map[string]string)

	var merr *multierror.Error
	for name, def := range definitions {
		envValue, exists := os.LookupEnv(v1beta1.ParameterToEnvironmentVariable(name))
		if def.Required && !exists {
			merr = multierror.Append(merr, fmt.Errorf("parameter %s is required", name))
			continue
		}
		value := def.Default
		if exists {
			value = envValue
		}
		params[name] = value
	}

	return params, merr.ErrorOrNil()
}
