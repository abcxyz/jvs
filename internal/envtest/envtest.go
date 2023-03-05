// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envtest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/abcxyz/jvs/assets"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/cfgloader"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/renderer"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// ServerConfigResponse is the response from creating a server config.
type ServerConfigResponse struct {
	Config    *config.UIServiceConfig
	Renderer  *renderer.Renderer
	Processor *justification.Processor
}

// BuildFormRequest builds an http request and http response recorder for the
// given form values (expressed as url.Values). It sets the proper headers and
// response types to post as a form and expect HTML in return.
func BuildFormRequest(ctx context.Context, tb testing.TB, meth, pth string, v *url.Values) (*httptest.ResponseRecorder, *http.Request) {
	tb.Helper()

	var body io.Reader
	if v != nil {
		body = strings.NewReader(v.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, meth, pth, body)
	if err != nil {
		tb.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "/back")
	return httptest.NewRecorder(), req
}

// NewServerConfig creates a new server configuration. It creates all the keys,
// databases, and cacher, but does not actually start the server. All cleanup is
// scheduled by t.Cleanup.
func NewServerConfig(tb testing.TB, port string, allowlist []string, devMode bool) *ServerConfigResponse {
	tb.Helper()

	logger := logging.TestLogger(tb)
	ctx := logging.WithLogger(context.Background(), logger)

	uiCfg := &config.UIServiceConfig{
		Port:      port,
		Allowlist: allowlist,
		DevMode:   devMode,
	}

	// Create the renderer.
	r, err := renderer.New(ctx, assets.ServerFS(),
		renderer.WithDebug(true),
		renderer.WithOnError(func(err error) {
			logger.Error(err)
		}))
	if err != nil {
		tb.Fatal(err)
	}

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	version := key + "/cryptoKeyVersions/[VERSION]"

	// Mock KMS.
	mockKMS := testutil.NewMockKeyManagementServer(key, version, jvscrypto.PrimaryLabelPrefix+"[VERSION]"+"-0")

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKMS)

	_, conn := pkgtestutil.FakeGRPCServer(tb, func(s *grpc.Server) {
		kmspb.RegisterKeyManagementServiceServer(s, mockKMS)
	})

	opt := option.WithGRPCConn(conn)
	tb.Cleanup(func() {
		conn.Close()
	})

	kmsClient, err := kms.NewKeyManagementClient(ctx, opt)
	if err != nil {
		tb.Fatal(err)
	}

	var cfg config.JustificationConfig
	if err := cfgloader.Load(ctx, &cfg); err != nil {
		tb.Fatal(err)
	}

	p := justification.NewProcessor(kmsClient, &cfg)

	return &ServerConfigResponse{
		Config:    uiCfg,
		Renderer:  r,
		Processor: p,
	}
}
