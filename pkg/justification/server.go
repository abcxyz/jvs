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

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// JVSAgent is the implementation of the justification verification server.
type JVSAgent struct {
	jvspb.JVSServiceServer

	Processor *Processor
}

// NewJVSAgent creates a new JVSAgent.
func NewJVSAgent(p *Processor) *JVSAgent {
	// gcpjwt.SigningMethodKMSES256.Override()
	return &JVSAgent{Processor: p}
}

func (j *JVSAgent) CreateJustification(ctx context.Context, req *jvspb.CreateJustificationRequest) (*jvspb.CreateJustificationResponse, error) {
	token, err := j.Processor.CreateToken(ctx, req)
	if err != nil {
		return nil, err
	}

	return &jvspb.CreateJustificationResponse{
		Token: token,
	}, nil
}
