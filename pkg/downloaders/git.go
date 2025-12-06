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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/crashappsec/ocular-default-integrations/internal/definitions"
	"github.com/crashappsec/ocular-default-integrations/internal/utils"
	"github.com/crashappsec/ocular/api/v1beta1"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	format "github.com/go-git/go-git/v6/plumbing/format/config"
	"github.com/go-git/go-git/v6/plumbing/transport"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Git struct{}

type GitMetadata struct {
	Ref      string `json:"ref,omitempty"`
	Hash     string `json:"hash,omitempty"`
	CloneURL string `json:"clone_url,omitempty"`
	Public   bool   `json:"public,omitempty"`
}

const CustomScope = "/etc/ocular/gitconfig"

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
			MountPath: CustomScope,
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

	cfg, err := repo.Config()
	if err != nil {
		return err
	}
	cfg.Raw.SetOption("core", "", "sharedRepository", "all")
	cfg.Core.RepositoryFormatVersion = format.Version_0

	if f, err := os.Stat(CustomScope); err == nil && !f.IsDir() {
		l.Info("applying custom git config", "path", CustomScope)
		f, err := os.Open(CustomScope)
		if err != nil {
			l.Error(err, "failed to open custom git config", "path", CustomScope)
		} else {
			customCfg, err := config.ReadConfig(f)
			if err != nil {
				l.Error(err, "failed to read custom git config", "path", CustomScope)
			} else {
				cfg = ptr.To(config.Merge(cfg, customCfg))
			}
		}
	}

	err = repo.SetConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to set custom git config: %w", err)
	}

	// Add remote and fetch
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
		Fetch: []config.RefSpec{
			"+HEAD:refs/remotes/origin/HEAD",
			"+refs/*:refs/*",
		},
	})
	if err != nil {
		return err
	}

	err = repo.FetchContext(ctx, &gogit.FetchOptions{
		Progress: utils.NewLogWriter(l),
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

	metadata := GitMetadata{
		CloneURL: cloneURL,
	}

	checkoutOptions, err := getGitCheckoutOption(ctx, repo, version)
	if err != nil {
		return err
	}

	metadata.Ref = checkoutOptions.Branch.String()
	metadata.Hash = checkoutOptions.Hash.String()

	l.Info("checking out revision", "ref", checkoutOptions.Branch, "hash", checkoutOptions.Hash)

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	if err = worktree.Checkout(checkoutOptions); err != nil {
		return err
	}

	if err = writeJSONStruct(GitMetadataPath, metadata); err != nil {
		l.Error(err, "failed to write git metadata")
	}

	// TODO(bryce): This is due to go-git creating object files without respecting sharedRepository
	// This needs to be fixed in go-git
	// See: https://github.com/go-git/go-git/issues/1572
	err = filepath.WalkDir(".git/objects", chmodRecursive)
	if err != nil {
		l.Error(err, "failed to set permissions on .git directory")
	}
	return nil
}

func chmodRecursive(path string, e fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if e.IsDir() {
		return nil
	}

	return os.Chmod(path, 0o644)
}

func getGitCheckoutOption(ctx context.Context, repo *gogit.Repository, version string) (*gogit.CheckoutOptions, error) {
	l := log.FromContext(ctx)
	var (
		checkoutOptions *gogit.CheckoutOptions
		ref             *plumbing.Reference
		err             error
	)

	switch {
	case version == "":
		ref, err = repo.Reference(plumbing.NewRemoteHEADReferenceName("origin"), true)
		if err != nil {
			l.Info("failed to find HEAD ref, using default branch")
			return &gogit.CheckoutOptions{
				Branch: plumbing.NewRemoteReferenceName("origin", "main"),
			}, nil
		}
	case plumbing.IsHash(version):
		ref = plumbing.NewHashReference("", plumbing.NewHash(version))
	default:
		ref, err = repo.Reference(plumbing.NewRemoteReferenceName("origin", version), false)
		if err != nil {
			return nil, err
		}
	}
	l.Info("resolved reference", "type", ref.Type(), "name", ref.Name(), "hash", ref.Hash(), "target", ref.Target())

	switch ref.Type() {
	case plumbing.SymbolicReference:
		checkoutOptions = &gogit.CheckoutOptions{
			Branch: ref.Target(),
		}
	case plumbing.HashReference:
		checkoutOptions = &gogit.CheckoutOptions{
			Hash: ref.Hash(),
		}
	default:
		return nil, fmt.Errorf("unsupported reference type: %s", ref.Type())
	}

	return checkoutOptions, nil
}

const GitMetadataPath = v1beta1.PipelineMetadataDirectory + "/git.json"

func (g Git) GetMetadataFiles() []string {
	return []string{
		GitMetadataPath,
	}
}
