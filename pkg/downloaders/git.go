// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package downloaders

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Git struct{}

var _ Downloader = Git{}

func (Git) GetName() string {
	return "git"
}

func (Git) GetEnvSecrets() []definitions.EnvironmentSecret {
	return nil
}

func (Git) GetFileSecrets() []definitions.FileSecret {
	return []definitions.FileSecret{
		{
			SecretKey: "gitconfig",
			MountPath: "/etc/gitconfig",
		},
	}
}

func (Git) EnvironmentVariables() []corev1.EnvVar {
	return nil
}

func (Git) Download(ctx context.Context, cloneURL, version, targetDir string) error {
	l := log.FromContext(ctx).WithValues("cloneURL", cloneURL, "targetDir", targetDir)

	// Initialize empty local repo
	repo, err := gogit.PlainInit(targetDir, false)
	if err != nil {
		return err
	}

	repoCfg, err := repo.Config()
	if err != nil {
		return err
	}

	// Parse /etc/gitconfig
	cfg, err := config.LoadConfig(config.SystemScope)
	if err != nil {
		l.Info("failed to load config - ignore this if no config was set", "error", err)
	} else {
		cfg.Core = repoCfg.Core

		if err := repo.Storer.SetConfig(cfg); err != nil {
			return err
		}
	}

	// Add remote and fetch
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
	})
	if err != nil {
		return err
	}

	err = repo.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			"+HEAD:refs/remotes/origin/HEAD",
			"+refs/heads/*:refs/remotes/origin/*",
		},
		Progress: os.Stderr,
	})
	switch {
	case errors.Is(err, gogit.NoErrAlreadyUpToDate):
		l.Info("repository already up to date")
	case errors.Is(err, transport.ErrEmptyRemoteRepository):
		l.Info("repository is empty, nothing to fetch")
		return nil
	case err != nil:
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	l.Info("cloned Git repository")

	var checkoutOptions *gogit.CheckoutOptions
	switch {
	case version == "":
		ref, err := repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
		if err != nil {
			l.Error(err, "failed to resolve Git HEAD ref, defaulting to main")
			checkoutOptions = &gogit.CheckoutOptions{
				Branch: plumbing.NewRemoteReferenceName("origin", "main"),
			}
		} else {
			l.Info("resolved Git HEAD ref", "ref", ref.Name().String())
			checkoutOptions = &gogit.CheckoutOptions{
				Branch: ref.Name(),
			}
		}
	case plumbing.IsHash(version):
		l = l.WithValues("hash", version)
		checkoutOptions = &gogit.CheckoutOptions{
			Hash: plumbing.NewHash(version),
		}
	default:
		l = l.WithValues("branch", version)
		checkoutOptions = &gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(version),
		}
	}
	l.Info("checking out revision")

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	return worktree.Checkout(checkoutOptions)
}
