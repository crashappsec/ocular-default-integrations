#!/usr/bin/env bash
# Copyright (C) 2025 Crash Override, Inc.
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the FSF, either version 3 of the License, or (at your option) any later version.
# See the LICENSE file in the root of this repository for full license text or
# visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

SCRIPT_DIRECTORY=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
ROOT_DIRECTORY=$(readlink -f "$SCRIPT_DIRECTORY/../../")

set -e

CHART_DIRECTORY="${ROOT_DIRECTORY}/dist/chart"

mkdir -p "$CHART_DIRECTORY"

if [ ! -f "$CHART_DIRECTORY/Chart.yaml" ]; then
  cat >"$CHART_DIRECTORY/Chart.yaml" <<EOF
apiVersion: v2
name: ocular-default-integrations
description: A Helm chart for deploying default ocular integrations
type: application
version: 0.1.0
appVersion: "${OCULAR_DEFAULTS_VERSION:-latest}"
EOF
else
  yq -ie '.appVersion = (strenv(OCULAR_DEFAULTS_VERSION) | sub("^v", ""))' "$CHART_DIRECTORY/Chart.yaml"
fi

if [ ! -f "$CHART_DIRECTORY/values.yaml" ]; then
  cat >"$CHART_DIRECTORY/values.yaml" <<EOF
crawlers:
  image:
    repository: "ghcr.io/crashappsec/ocular-default-crawlers"
    tag: "${OCULAR_DEFAULTS_VERSION:-latest}"
downloaders:
  image:
    repository: "ghcr.io/crashappsec/ocular-default-downloaders"
    tag: "${OCULAR_DEFAULTS_VERSION:-latest}"
uploaders:
  image:
    repository: "ghcr.io/crashappsec/ocular-default-uploaders"
    tag: "${OCULAR_DEFAULTS_VERSION:-latest}"
EOF
else
  yq -ie ".crawlers.image.tag = \"${OCULAR_DEFAULTS_VERSION:-latest}\"" "$CHART_DIRECTORY/values.yaml"
  yq -ie ".downloaders.image.tag = \"${OCULAR_DEFAULTS_VERSION:-latest}\"" "$CHART_DIRECTORY/values.yaml"
  yq -ie ".uploaders.image.tag = \"${OCULAR_DEFAULTS_VERSION:-latest}\"" "$CHART_DIRECTORY/values.yaml"
fi
resource_kinds=("crawlers" "downloaders" "uploaders")

for kind in "${resource_kinds[@]}"; do
  kind_templates_dir="$CHART_DIRECTORY/templates/$kind"
  mkdir -p "$kind_templates_dir"
  (cd "$kind_templates_dir" && "${ROOT_DIRECTORY}/bin/kustomize" build "$ROOT_DIRECTORY/config/$kind" | yq ".spec.container.image = \"{{ .Values.$kind.image.repository }}:{{ .Values.$kind.image.tag }}\"" -s '.metadata.name + ".yaml"')
done



