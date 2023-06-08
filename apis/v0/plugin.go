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

package v0

import (
	"context"
	"fmt"
)

// This will be a plugin interface for go-plugin.
// The input/output should be adjusted in real case.
type PluginVerifier interface {
	// The real verifier should probably return more meaningful things in addition
	// to the potential error.
	Verify(context.Context, *Justification) error
}

// The location of this func should be reconsidered.
func InitPlugins(ctx context.Context, plugins []string) (map[string]PluginVerifier, error) {
	// This func should use go-plugin to initialize the plugins for real.
	return map[string]PluginVerifier{
		"fake": &fakeVerifier{},
	}, nil
}

type fakeVerifier struct{}

func (v *fakeVerifier) Verify(_ context.Context, _ *Justification) error {
	return fmt.Errorf("fake verifier: nothing implemented")
}
