// Copyright 2023 Google LLC
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

package plugin

import (
	"context"
	"fmt"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// GRPCClient is an implementation of Validator that talks over RPC.
type GRPCClient struct {
	client jvspb.JVSPluginClient
}

func (m *GRPCClient) Validate(justification *jvspb.Justification) (*jvspb.ValidateJustificationResponse, error) {
	resp, err := m.client.Validate(context.Background(), &jvspb.ValidateJustificationRequest{
		Justification: justification,
	})

	if err != nil {
		return resp, fmt.Errorf("failed to validate justification: %w", err)
	}
	return resp, nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	jvspb.JVSPluginServer
	// This is the real implementation
	Impl Validator
}

func (m *GRPCServer) Validate(
	ctx context.Context,
	req *jvspb.ValidateJustificationRequest) (*jvspb.ValidateJustificationResponse, error) {
	resp, err := m.Impl.Validate(ctx, req.Justification)

	if err != nil {
		return resp, fmt.Errorf("failed to validate justification: %w", err)
	}
	return resp, nil
}
