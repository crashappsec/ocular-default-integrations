# Copyright (C) 2025 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

FROM golang:1.25@sha256:5502b0e56fca23feba76dbc5387ba59c593c02ccc2f0f7355871ea9a0852cebe AS builder
ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS="-w -s"
# INTEGRATION should be set to one of the following:
# - downloaders
# - uploaders
# - crawlers
ARG INTEGRATION


WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY internal/ internal/
COPY pkg/ pkg/
COPY /cmd/default-${INTEGRATION}/ cmd/


RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -ldflags="$LDFLAGS" -trimpath -o entrypoint cmd/main.go

FROM gcr.io/distroless/static:nonroot@sha256:a9f88e0d99c1ceedbce565fad7d3f96744d15e6919c19c7dafe84a6dd9a80c61

WORKDIR /
COPY --from=builder /workspace/entrypoint /bin/entrypoint
USER 65538:65538

LABEL org.opencontainers.image.source="https://github.com/crashappsec/ocular-default-integrations"

ENTRYPOINT ["/bin/entrypoint"]
