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

package v0

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestExplanationValidator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		req      *ValidateJustificationRequest
		wantResp *ValidateJustificationResponse
	}{
		{
			name: "success",
			req: &ValidateJustificationRequest{
				Justification: &Justification{
					Category: "explanation",
					Value:    "I have reasons",
				},
			},
			wantResp: &ValidateJustificationResponse{
				Valid: true,
			},
		},
		{
			name: "success",
			req: &ValidateJustificationRequest{
				Justification: &Justification{
					Category: "explanation",
				},
			},
			wantResp: &ValidateJustificationResponse{
				Valid: false,
				Error: []string{"explanation cannot be empty"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotResp, err := DefaultJustificationValidator.Validate(t.Context(), tc.req)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.wantResp, gotResp, protocmp.Transform()); diff != "" {
				t.Errorf("Validation response (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestGetUIDataInValidator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		req      *GetUIDataRequest
		wantResp *UIData
	}{
		{
			name: "success",
			req:  &GetUIDataRequest{},
			wantResp: &UIData{
				DisplayName: "Explanation",
				Hint:        "A justification reason in free-form text.",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotResp, err := DefaultJustificationValidator.GetUIData(t.Context(), tc.req)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.wantResp, gotResp, protocmp.Transform()); diff != "" {
				t.Errorf("GetUIData response (-want,+got):\n%s", diff)
			}
		})
	}
}
