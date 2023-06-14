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
	"fmt"
	"os/exec"
	"path/filepath"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/pkg/multicloser"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
)

const (
	// PluginGlob is the glob pattern used to find plugins.
	PluginGlob = "jvs-plugin-*"
)

// PluginManager will discover and load plugins.
type PluginManager struct {
	// dir is the directory where plugins can be found.
	dir string
}

// LoadPlugins loads plugins in the dir and put them into the Validator interface.
func (p *PluginManager) LoadPlugins() (map[string]jvspb.Validator, *multicloser.Closer, error) {
	validators := make(map[string]jvspb.Validator)
	var merr *multierror.Error
	var closer *multicloser.Closer

	// Load from the dir.
	paths, err := plugin.Discover(PluginGlob, p.dir)
	if err != nil {
		return validators, nil, fmt.Errorf(
			"error discovering plugins in %s: %w", p.dir, err)
	}

	for _, path := range paths {
		name := filepath.Base(path)

		// Enable the plugin.
		pluginClient := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  jvspb.Handshake,
			Cmd:              exec.Command(path),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		})

		// Connect plugin via RPC.
		rpcClient, err := pluginClient.Client()
		if err != nil {
			merr = multierror.Append(fmt.Errorf("failed to initiate plugin client %s : %w", name, err))
			break
		}
		closer = multicloser.Append(closer, rpcClient.Close)

		// Request the plugin.
		raw, err := rpcClient.Dispense(name)
		if err != nil {
			merr = multierror.Append(fmt.Errorf("failed to dispense plugin %s : %w", name, err))
			break
		}

		v, ok := raw.(jvspb.Validator)
		if !ok {
			merr = multierror.Append(fmt.Errorf("failed to cast plugin %s to Validator interface", name))
			break
		}
		validators[name] = v
	}
	return validators, closer, merr.ErrorOrNil()
}
