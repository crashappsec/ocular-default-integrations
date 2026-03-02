# Copyright (C) 2025-2026 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

FROM golang:1.26@sha256:9edf71320ef8a791c4c33ec79f90496d641f306a91fb112d3d060d5c1cee4e20 AS builder
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

FROM gcr.io/distroless/static:nonroot@sha256:f512d819b8f109f2375e8b51d8cfd8aafe81034bc3e319740128b7d7f70d5036

WORKDIR /
COPY --from=builder /workspace/entrypoint /entrypoint
USER 65538:65538

LABEL org.opencontainers.image.source="https://github.com/crashappsec/ocular-default-integrations"

ENTRYPOINT ["/entrypoint"]
