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

// Package plugin provides functions to manage plugins.
package plugin

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/pkg/multicloser"
	"github.com/hashicorp/go-plugin"
)

const (
	// PluginGlob is the glob pattern used to find plugins.
	PluginGlob = "jvs-plugin-*"
)

// LoadPlugins loads plugins in the dir and put them into the Validator interface.
func LoadPlugins(dir string) (map[string]jvspb.Validator, *multicloser.Closer, error) {
	validators := make(map[string]jvspb.Validator)
	var merr error
	var closer *multicloser.Closer

	// Load from the dir.
	paths, err := plugin.Discover(PluginGlob, dir)
	if err != nil {
		return validators, nil, fmt.Errorf(
			"error discovering plugins in %s: %w", dir, err)
	}

	for _, path := range paths {
		// PluginGlob prefix won't be part of the name.
		prefix := len(PluginGlob) - 1
		name := filepath.Base(path)[prefix:]

		// Enable the plugin.
		pluginClient := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  jvspb.Handshake,
			Cmd:              exec.Command(path),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		})
		closer = multicloser.Append(closer, pluginClient.Kill)

		// Connect plugin via RPC.
		rpcClient, err := pluginClient.Client()
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to initiate plugin client %s : %w", name, err))
			continue
		}

		// Request the plugin.
		raw, err := rpcClient.Dispense(name)
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to dispense plugin %s : %w", name, err))
			continue
		}

		v, ok := raw.(jvspb.Validator)
		if !ok {
			merr = errors.Join(merr, fmt.Errorf("failed to cast plugin %s to Validator interface", name))
			continue
		}
		validators[name] = v
	}
	return validators, closer, merr
}
