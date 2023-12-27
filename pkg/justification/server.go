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

package justification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abcxyz/jvs/apis/v0/v0connect"
	"github.com/lestrrat-go/jwx/v2/jwt"
	grpcmetadata "google.golang.org/grpc/metadata"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// JVSAgent is the implementation of the justification verification server.
type JVSAgent struct {
	v0connect.JVSServiceHandler

	Processor *Processor
}

// NewJVSAgent creates a new JVSAgent.
func NewJVSAgent(p *Processor) *JVSAgent {
	return &JVSAgent{Processor: p}
}

func (j *JVSAgent) CreateJustification(ctx context.Context, req *jvspb.CreateJustificationRequest) (*jvspb.CreateJustificationResponse, error) {
	requestor, err := extractRequestorFromIncomingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract request principal: %w", err)
	}

	token, err := j.Processor.CreateToken(ctx, requestor, req)
	if err != nil {
		return nil, err
	}

	return &jvspb.CreateJustificationResponse{
		Token: string(token),
	}, nil
}

// extractRequestorFromIncomingContext attempts to extract the callers identity
// from the incoming authentication context. Right now, it assumes Google Cloud
// IAP or Google CLoud Run identity tokens, but could be extended to support
// other lookups in the future.
//
// It returns the empty string if there's no authentication metadata in the
// incoming context. It returns an error if there is data, but it is malformed
// or unparseable.
func extractRequestorFromIncomingContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", nil
	}

	md, ok := grpcmetadata.FromIncomingContext(ctx)
	if !ok {
		return "", nil
	}

	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", nil
	}

	raw := strings.TrimSpace(vals[0])
	if len(raw) < 8 {
		return "", fmt.Errorf("invalid jwt in grpc metadata (too short)")
	}
	raw = raw[7:] // strip "bearer "

	// We intentionally do not validate or verify the token since the upstream
	// authentication should have taken care of it. In the case of Cloud Run, the
	// signature is stripped, so we can't even verify if we wanted.
	t, err := jwt.ParseInsecure([]byte(raw),
		jwt.WithAcceptableSkew(5*time.Second),
	)
	if err != nil {
		return "", fmt.Errorf("failed to parse incoming jwt: %w", err)
	}

	principalRaw, ok := t.Get("email")
	if !ok {
		return "", fmt.Errorf(`missing "email" key in incoming jwt`)
	}
	principal, ok := principalRaw.(string)
	if !ok {
		return "", fmt.Errorf(`"email" key is not of type %T (got %T)`, "", principalRaw)
	}
	return principal, nil
}
