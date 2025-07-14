# Copyright (C) 2025 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

FROM golang:1.24.4-alpine@sha256:ddf52008bce1be455fe2b22d780b6693259aaf97b16383b6372f4b22dd33ad66 AS builder

WORKDIR /app

RUN apk add --no-cache \
    git \
    make \
    ca-certificates \
    && update-ca-certificates

ARG LDFLAGS="-s -w"
# INTEGRATION should be set to one of the following:
# - downloaders
# - uploaders
# - crawlers
ARG INTEGRATION

COPY go.mod go.sum /app/

ENV GOPRIVATE=github.com/crashappsec/ocular
RUN --mount=type=secret,id=netrc,target=/root/.netrc \
    --mount=type=cache,target=/go/pkg/mod go mod download

COPY /cmd/default-${INTEGRATION}/ /app/cmd/default-${INTEGRATION}/
COPY /internal /app/internal
COPY /pkg /app/pkg

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags="$LDFLAGS" -o /app/entrypoint /app/cmd/default-${INTEGRATION}/main.go

FROM alpine:3.22@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715

COPY --from=builder /app/entrypoint /bin/entrypoint

LABEL org.opencontainers.image.source="https://github.com/crashappsec/ocular-default-integrations"

ENTRYPOINT ["/bin/entrypoint"]
