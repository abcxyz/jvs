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

	"github.com/sethvargo/go-envconfig"
)

type config struct {
	APIServer         string `env:"INTEG_TEST_API_SERVER,required"`
	APIServiceIDToken string `env:"INTEG_TEST_API_SERVICE_ID_TOKEN,required"`
	UIServiceIDToken  string `env:"INTEG_TEST_UI_SERVICE_ID_TOKEN,required"`
	JWKSEndpoint      string `env:"INTEG_TEST_JWKS_ENDPOINT,required"`
	ServiceAccount    string `env:"INTEG_TEST_WIF_SERVICE_ACCOUNT,required"`
	UIServiceAddr     string `env:"INTEG_TEST_UI_SERVICE_ADDR,required"`
}

func newTestConfig(ctx context.Context) (*config, error) {
	var c config
	if err := envconfig.ProcessWith(ctx, &c, envconfig.OsLookuper()); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}
	return &c, nil
}
