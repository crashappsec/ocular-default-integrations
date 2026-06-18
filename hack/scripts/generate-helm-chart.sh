#!/usr/bin/env bash
# Copyright (C) 2025-2026 Crash Override, Inc.
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
appVersion="${OCULAR_DEFAULTS_VERSION:-latest}"

cat >"$CHART_DIRECTORY/Chart.yaml" <<EOF
apiVersion: v2
name: ocular-default-integrations
version: "${OCULAR_DEFAULTS_HELM_VERSION:-0.0.0}"
appVersion: "${appVersion#v}"
kubeVersion: ">=1.26.0-0"
description: A Helm chart for deploying default ocular integrations
type: application
icon: "https://ocularproject.io/favicon-32x32.png"
home: https://ocularproject.io
sources:
  - https://github.com/crashappsec/ocular-default-integrations
maintainers:
  - name: Bryce Thuiot
    email: bryce@crashoverride.com
keywords:
  - git
  - kubernetes
  - scanning
  - sast
  - secrets
  - sca
  - security
annotations:
  artifacthub.io/category: security
  artifacthub.io/license: GPL-3.0
EOF


cat >"$CHART_DIRECTORY/values.yaml" <<EOF
# configuration of crawlers
crawlers:
  secretName: "crawler-secrets"
  image:
    repository: "ghcr.io/crashappsec/ocular-default-crawlers"
    tag: "v{{ .Chart.AppVersion }}"

# configuration of downloaders
downloaders:
  secretName: "downloader-secrets"
  image:
    repository: "ghcr.io/crashappsec/ocular-default-downloaders"
    tag: "v{{ .Chart.AppVersion }}"

# Configuration for uploaders
uploaders:
  secretName: "uploader-secrets"
  image:
    repository: "ghcr.io/crashappsec/ocular-default-uploaders"
    tag: "v{{ .Chart.AppVersion }}"
EOF
 
resource_kinds=("crawler" "downloader" "uploader")

for kind in "${resource_kinds[@]}"; do
    kind_templates_dir="$CHART_DIRECTORY/templates/${kind}s"
    # create template directory for kind
    mkdir -p "$kind_templates_dir"
    # Generate the templates, then have yq edit them
    # and split them into individual files
    (cd "$kind_templates_dir" && "${ROOT_DIRECTORY}/bin/kustomize" build "$ROOT_DIRECTORY/config/${kind}s" \
	     | sed -e "s/${kind}-secrets/\"{{ \$values.${kind}s.secretName }}\"/g" \
	     | yq ".spec.container.image = \"{{ \$values.${kind}s.image.repository }}:{{ \$values.${kind}s.image.tag }}\"" -s '.metadata.name + ".yaml"')
    # Add the "values templating" header trick to each file
    for f in "$kind_templates_dir"/*.yaml; do
	cat > "$f.tmp" <<'EOF'
{{- $values := (tpl (.Values | toYaml) $) | fromYaml }}
{{- $values := (tpl ($values | toYaml) $) | fromYaml }}
---
EOF
    cat "$f" >> "$f.tmp" && mv "$f.tmp" "$f"
    done
done



