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

package idtoken

import (
	"context"
	"os"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Log in as an end user with gcloud.
// `gcloud auth application-default login`
// Set env var MANUAL_TEST=true to run the test.
func TestFromDefaultCredentials(t *testing.T) {
	if os.Getenv("MANUAL_TEST") == "" {
		t.Skip("Skip manual test; set env var MANUAL_TEST to enable")
	}

	ts, err := FromDefaultCredentials(context.Background(), DefaultGoogleConfig)
	if err != nil {
		t.Errorf("failed to get ID token source: %v", err)
	}

	tk, err := ts.Token()
	if err != nil {
		t.Errorf("failed to get ID token: %v", err)
	}

	if _, err := jwt.Parse([]byte(tk.AccessToken), jwt.WithVerify(false)); err != nil {
		t.Errorf("%q not a valid ID token: %v", tk.AccessToken, err)
	}
}
