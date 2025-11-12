// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package dockerhub

import (
	"context"
	"fmt"
	"time"
)

type Repository struct {
	Name              string         `json:"name"`
	Namespace         string         `json:"namespace"`
	RepositoryType    RepositoryType `json:"repository_type,omitempty"`
	Status            int            `json:"status"`
	StatusDescription string         `json:"status_description"`
	Description       string         `json:"description"`
	IsPrivate         bool           `json:"is_private"`
	StarCount         int            `json:"star_count"`
	PullCount         int            `json:"pull_count"`
	LastUpdated       time.Time      `json:"last_updated"`
	LastModified      time.Time      `json:"last_modified"`
	DateRegistered    time.Time      `json:"date_registered"`
	Affiliation       string         `json:"affiliation"`
	MediaTypes        []string       `json:"media_types"`
	ContentTypes      []string       `json:"content_types"`
	Categories        []string       `json:"categories"`
	StorageSize       int64          `json:"storage_size"`
}

func (c *client) ListNamespaceRepositories(ctx context.Context, namespace string) ([]Repository, error) {
	u, err := buildURL("/namespaces/"+namespace+"/repositories/", map[string]string{
		"page_size": "100",
	})
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}
	return makePaginatedGetRequest[Repository](ctx, c, u)
}

type Tag struct {
	Id     int `json:"id"`
	Images []struct {
		Architecture string `json:"architecture"`
		Features     string `json:"features"`
		Variant      string `json:"variant"`
		Digest       string `json:"digest"`
		Layers       []struct {
			Digest      string `json:"digest"`
			Size        int    `json:"size"`
			Instruction string `json:"instruction"`
		} `json:"layers"`
		Os         string    `json:"os"`
		OsFeatures string    `json:"os_features"`
		OsVersion  string    `json:"os_version"`
		Size       int       `json:"size"`
		Status     string    `json:"status"`
		LastPulled time.Time `json:"last_pulled"`
		LastPushed time.Time `json:"last_pushed"`
	} `json:"images"`
	Creator             int       `json:"creator"`
	LastUpdated         time.Time `json:"last_updated"`
	LastUpdater         int       `json:"last_updater"`
	LastUpdaterUsername string    `json:"last_updater_username"`
	Name                string    `json:"name"`
	Repository          int       `json:"repository"`
	FullSize            int       `json:"full_size"`
	V2                  bool      `json:"v2"`
	Status              string    `json:"status"`
	TagLastPulled       time.Time `json:"tag_last_pulled"`
	TagLastPushed       time.Time `json:"tag_last_pushed"`
}

func (c *client) ListRepositoryTags(ctx context.Context, namespace, repository string) ([]Tag, error) {
	u, err := buildURL("/namespaces/"+namespace+"/repositories/"+repository+"/tags", map[string]string{
		"page_size": "100",
	})
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}
	return makePaginatedGetRequest[Tag](ctx, c, u)
}
