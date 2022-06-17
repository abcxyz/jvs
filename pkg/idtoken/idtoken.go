// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package idtoken provides functions to generate id tokens for end users.
package idtoken

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// FromDefaultCredentials creates a token source with the application default credentials (ADC).
// https://developers.google.com/accounts/docs/application-default-credentials
// It only works when the application default credentials is of an end user.
// Typically it's done with `gcloud auth application-default login`.
func FromDefaultCredentials(ctx context.Context) (oauth2.TokenSource, error) {
	ts, err := google.DefaultTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find google default credential: %w", err)
	}

	return oauth2.ReuseTokenSource(nil, &tokenSource{
		tokenSource: ts,
	}), nil
}

type tokenSource struct {
	tokenSource oauth2.TokenSource
}

// Token extracts the id_token field from ADC from a default token source and
// puts the value into the AccessToken field.
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("missing id_token")
	}

	return &oauth2.Token{
		AccessToken: idToken,
		Expiry:      token.Expiry,
	}, nil
}
