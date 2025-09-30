<br />
<div align="center">
    <h1 align="center">
        Ocular Default Integrations
    </h1>

  <p align="center">
        A collection of default integrations for Ocular, including uploaders, downloaders, and crawlers.
        These integrations are designed to be used with the Ocular API to solve common use cases.
  </p>
</div>

<hr/>

[![Documentation Site](https://img.shields.io/badge/docs-ocularproject.io-blue)](https://ocularproject.io/docs/)
[![Go Report Card](https://goreportcard.com/badge/github.com/crashappsec/ocular-default-integrations)](https://goreportcard.com/report/github.com/crashappsec/ocular-default-integrations)
[![GitHub Release](https://img.shields.io/github/v/release/crashappsec/ocular-default-integrations)](https://github.com/crashappsec/ocular-default-integrations/releases)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

A collection of default uploaders, downloaders, and crawlers that can be used to quickly get started with Ocular.

These integrations are included in the helm
chart "[Ocular default integrations](https://artifacthub.io/packages/helm/crashoverride-helm-charts/ocular-default-integrations)".

## Installation

Ensure that the [Ocular chart](https://artifacthub.io/packages/helm/crashoverride-helm-charts/ocular) is installed and
configured.
Then, install the default integrations chart:

```bash
helm repo add crashoverride-helm-charts https://crashoverride-helm-charts.storage
helm repo update

# Should be the namespace you want to run pipelines/searches in
NAMESPACE="ocular"

helm install ocular-default-integrations crashoverride-helm-charts/ocular-default-integrations \
    --namespace $NAMESPACE \
    --create-namespace
# Resource will then be available as a CRD in the cluster
# kubectl get crawlers -n $NAMESPACE
# kubectl get downloaders -n $NAMESPACE
# kubectl get uploaders -n $NAMESPACE
```
