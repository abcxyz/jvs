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
	"os"
	"os/exec"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"
)

// TODO: move to config folder.
type PluginConfig struct {
	Name       string
	EntryPoint string

	// Optional checksum.
	Checksum string
}

func InitValidators(ctx context.Context, configs []PluginConfig) (map[string]Validator, error) {
	validators := make(map[string]Validator)
	var merr *multierror.Error
	for _, c := range configs {
		// Enable the plugin.
		pluginClient := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  Handshake,
			Plugins:          PluginMap,
			Cmd:              exec.Command("sh", "-c", os.Getenv(c.EntryPoint)),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		})

		// Connect plugin via RPC.
		rpcClient, err := pluginClient.Client()
		if err != nil {
			merr = multierror.Append(fmt.Errorf("failed to initiate plugin client %s : %w", c.Name, err))
			break
		}

		// Request the plugin.
		raw, err := rpcClient.Dispense(c.Name)
		if err != nil {
			merr = multierror.Append(fmt.Errorf("failed to request plugin %s : %w", c.Name, err))
			break
		}

		v, ok := raw.(Validator)
		if !ok {
			merr = multierror.Append(fmt.Errorf("failed to cast plugin %s to Validator interface", c.Name))
			break
		}
		validators[c.Name] = v
	}
	return validators, merr.ErrorOrNil()
}
