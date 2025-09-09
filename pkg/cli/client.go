// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/crashappsec/ocular/pkg/generated/clientset"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ParseKubernetesClientset(ctx context.Context) (*clientset.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)
	l := log.FromContext(ctx)

	if config, err = rest.InClusterConfig(); err != nil {

		l.Info("in-cluster configuration was unable to be parsed, trying kubeconfig")
		home := homedir.HomeDir()
		kubeconfigPath := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			l.Info("unable to build kubernetes config from flags, trying kubeconfig")
			return nil, fmt.Errorf("unable to parse in-cluster config and kubeconfig")
		}
	}

	cs, err := clientset.NewForConfig(config)
	return cs, err
}
