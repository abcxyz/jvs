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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// These configs are gcloud configs:
	// https://github.com/twistedpair/google-cloud-sdk/blob/master/google-cloud-sdk/lib/googlecloudsdk/core/config.py
	CloudSDKClientID          = "32555940559.apps.googleusercontent.com"
	CloudSDKClientNotSoSecret = "ZmssLNjJy2998hD4CTg2ejr2"
	GoogleOAuthTokenURL       = "https://oauth2.googleapis.com/token"
)

// Config is the config to generate id tokens.
type Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Audience     string
}

// DefaultGoogleConfig is the default config to generate id tokens.
// It uses the same client config as gcloud.
var DefaultGoogleConfig = &Config{
	ClientID:     CloudSDKClientID,
	ClientSecret: CloudSDKClientNotSoSecret,
	TokenURL:     GoogleOAuthTokenURL,
	Audience:     CloudSDKClientID,
}

// FromDefaultCredentials creates a token source with the application default credentials.
// https://developers.google.com/accounts/docs/application-default-credentials
// It only works when the application default credentials is of an end user.
// Typically it's done with `gcloud auth application-default login`.
func FromDefaultCredentials(ctx context.Context) (oauth2.TokenSource, error) {
	ts, err := google.DefaultTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find google default credential: %w", err)
	}

	return oauth2.ReuseTokenSource(nil, &tokenSource{
		refreshTokenSource: ts,
		cfg:                DefaultGoogleConfig,
	}), nil
}

type tokenSource struct {
	refreshTokenSource oauth2.TokenSource
	cfg                *Config
}

// Given a refresh token, generate an id token.
// For GCP, the client id and the audience must be in the same project.
//
// With FromDefaultCredentials, we reuse the refresh token from application default credentials.
// It uses the gcloud client id.
//
// TODO: Should our CLI support standalone 3-legged OAuth flow w/o relying on gcloud?
//
// For a full flow, reference: https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	rt, err := ts.refreshTokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	v := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {rt.RefreshToken},
		"client_id":     {ts.cfg.ClientID},
		"client_secret": {ts.cfg.ClientSecret},
		"audience":      {ts.cfg.Audience},
	}

	// Use the refresh token to exchange an id token.
	resp, err := http.DefaultClient.Post(ts.cfg.TokenURL, "application/x-www-form-urlencoded", strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// tokenRes is the JSON response body.
	// Interestingly, the actual id token is in its own field, but an oauth2.Token
	// only has an AccessToken field. As a result, we need convert it to an oauth2.Token.
	var tokenRes struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		IDToken     string `json:"id_token"`
		ExpiresIn   int64  `json:"expires_in"` // relative seconds from now
	}

	if err := json.Unmarshal(b, &tokenRes); err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: tokenRes.IDToken,
		TokenType:   tokenRes.TokenType,
		Expiry:      time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second),
	}, nil
}
