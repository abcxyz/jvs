// Copyright 2023 The Authors (see AUTHORS file)
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

package integration

import (
	"context"
	"fmt"
	"strings"

	"github.com/sethvargo/go-envconfig"
)

type config struct {
	AuthToken      string `env:"INTEG_TEAT_AUTH_TOKEN,required"`
	APIURL         string `env:"INTEG_TEST_API_URL,required"`
	UIURL          string `env:"INTEG_TEST_UI_URL,required"`
	PublicKeyURL   string `env:"INTEG_TEST_PUBLIC_KEY_URL,required"`
	CertRotatorURL string `env:"INTEG_TEST_CERT_ROTATOR_URL,required"`
	APISERVER      string
	JwksEndpoint   string
}

func newTestConfig(ctx context.Context) (*config, error) {
	var c config
	if err := envconfig.ProcessWith(ctx, &c, envconfig.OsLookuper()); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}
	c.APISERVER = fmt.Sprint(strings.Split(c.APIURL, "https://")[1], ":443")
	c.JwksEndpoint = fmt.Sprint(c.PublicKeyURL, "/.well-known/jwks")
	return &c, nil
}
