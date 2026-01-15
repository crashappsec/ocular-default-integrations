// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package dockerhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type RepositoryType string

const (
	RepositoryTypeImage  RepositoryType = "image"
	RepositoryTypePlugin RepositoryType = "plugin"
)

type Client interface {
	ListNamespaceRepositories(context.Context, string) ([]Repository, error)
	ListRepositoryTags(ctx context.Context, namespace, repository string) ([]Tag, error)
}

type client struct {
	authToken  string
	httpClient *http.Client
}

type Options struct {
	AuthToken string
}

func NewClient(options Options) Client {
	c := &client{
		httpClient: http.DefaultClient,
	}
	if options.AuthToken != "" {
		c.authToken = options.AuthToken
	}
	return c
}

const apiBaseURL = "https://hub.docker.com/v2/"

func buildURL(path string, queryParams map[string]string) (string, error) {
	baseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %w", err)
	}
	u := baseURL.JoinPath(path)

	q := u.Query()
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func makeRequest[Result any](ctx context.Context, c *client, method string, u string, body any) (Result, error) {
	var (
		bodyReader io.ReadCloser
		result     Result
	)
	if body != nil {
		// marshal body to json
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return result, fmt.Errorf("error marshaling request body: %w", err)
		}
		bodyReader = io.NopCloser(bytes.NewReader(jsonBody))
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return result, fmt.Errorf("error making request: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("received non-2xx response: %d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	return result, err
}

type PaginatedResponse[Result any] struct {
	Count    int      `json:"count"`
	Next     string   `json:"next"`
	Previous string   `json:"previous"`
	Results  []Result `json:"results"`
}

func makePaginatedGetRequest[Result any](ctx context.Context, c *client, u string) ([]Result, error) {
	var results []Result
	for {
		result, err := makeRequest[PaginatedResponse[Result]](ctx, c, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("error making request to endpoint '%s': %w", u, err)
		}
		results = append(results, result.Results...)
		if result.Next == "" {
			break
		}
		u = result.Next
	}
	return results, nil
}
