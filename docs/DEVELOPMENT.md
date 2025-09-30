# Development

This document describes the development process for the project.

## Getting Started

*NOTE*: This project is based on the standard [Kubebuilder](https://kubebuilder.io/) layout.
The following documenation is taken and adapted from the default README generated
by Kubebuilder. For more information, please refer to the [Kubebuilder documentation](https://kubebuilder.io/).

For a more in-depth guide on how to get started with the project and what the different components
are, please refer to the [documentation site](https://ocularproject.io/docs/)

### Prerequisites
- go
- docker
- kubectl
- Access to a Kubernetes v1.28.0+ cluster with [Ocular](https://ocularproject.io) installed.

*NOTE*: Any environment variable mentioned in the following commands can be set in the
`.env` file (or whatever file you set `OCULAR_ENV_FILE` to), which is loaded automatically by the `make` command.
An example `.env` file is provided in the repository as [`example.env`](/example.env).

### To Deploy on the cluster
**Build and push your images to the location specified by `OCULAR_UPLOADERS_IMG`, `OCULAR_DOWNLOADERS_IMG`, and `OCULAR_UPLOADERS_IMG`:**

```sh
make docker-build-all docker-push-all \
  OCULAR_UPLOADERS_IMG=<some-registry>/ocular-uploaders:tag \
  OCULAR_DOWNLOADERS_IMG=<some-registry>/ocular-downloaders:tag \
  OCULAR_CRAWLERS_IMG=<some-registry>/ocular-crawlers:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

### To Uninstall
**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer \
  OCULAR_UPLOADERS_IMG=<some-registry>/ocular-uploaders:tag \
  OCULAR_DOWNLOADERS_IMG=<some-registry>/ocular-downloaders:tag \
  OCULAR_CRAWLERS_IMG=<some-registry>/ocular-crawlers:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f ./dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

**NOTE**: This should only be used if you know what you are doing. Users should prefer installation from the [helm charts repository](https://github.com/crashappsec/helm-charts).

```sh
make build-helm \
  OCULAR_UPLOADERS_IMG=<some-registry>/ocular-uploaders:tag \
  OCULAR_DOWNLOADERS_IMG=<some-registry>/ocular-downloaders:tag \
  OCULAR_CRAWLERS_IMG=<some-registry>/ocular-crawlers:tag
```

2. See that a chart was generated under 'dist/chart', and users
   can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes.



