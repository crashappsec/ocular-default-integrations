// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package utils

import (
	"bufio"
	"bytes"
	"io"

	"github.com/go-logr/logr"
)

type LogWriter struct {
	logger logr.Logger
}

var _ io.Writer = (*LogWriter)(nil)

func NewLogWriter(l logr.Logger) *LogWriter {
	return &LogWriter{
		logger: l,
	}
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	reader := bytes.NewReader(p)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		lw.logger.Info(line)
	}
	return len(p), nil
}
