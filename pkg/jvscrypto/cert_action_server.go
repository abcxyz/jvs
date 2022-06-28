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

package jvscrypto

import (
	"context"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// JVSAgent is the implementation of the justification verification server.
type CertificateActionAgent struct {
	jvspb.CertificateActionServiceServer
	Service *CertificateActionService
}

// NewJVSAgent creates a new JVSAgent.
func NewCertificateActionAgent(s *CertificateActionService) *CertificateActionAgent {
	return &CertificateActionAgent{Service: s}
}

func (a *CertificateActionAgent) CertificateAction(ctx context.Context, req *jvspb.CertificateActionRequest) (*jvspb.CertificateActionResponse, error) {
	success, err := a.Service.CertificateAction(ctx, req)
	if err != nil {
		return nil, err
	}

	return &jvspb.CertificateActionResponse{
		Success: success,
	}, nil
}
