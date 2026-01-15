// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package input

import (
	"strings"
	"text/template"
)

// UserTemplater is a struct for parsing user-defined templates.
// This is done to provide a single point for any integrations that
// will template user input.
type UserTemplater struct {
	template *template.Template
}

func NewUserTemplater(name string) *UserTemplater {
	t := template.New(name)
	t.Funcs(template.FuncMap{
		"trimprefix": strings.TrimPrefix,
		"trimsuffix": strings.TrimSuffix,
		"tolower":    strings.ToLower,
		"toupper":    strings.ToUpper,
	})
	return &UserTemplater{
		template: t,
	}
}

func (ut *UserTemplater) Execute(userTemplate string, data any) (string, error) {
	var buf strings.Builder
	tmpl, err := ut.template.Parse(userTemplate)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
