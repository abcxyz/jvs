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

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

const (
	handshakeCookieKey   = "JVS_PLUGIN"
	handshakeCookieValue = "hello"
	JiraGRPCPlugin       = "jira_grpc_plugin"
)

// Handshake is a common handshake that is shared by plugin and host.
// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   handshakeCookieKey,
	MagicCookieValue: handshakeCookieValue,
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	JiraGRPCPlugin: &ValidatorGRPCPlugin{},
}

// The interface we are exposing as a plugin.
type Validator interface {
	Validate(context.Context, *jvspb.Justification) (*jvspb.ValidateJustificationResponse, error)
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type ValidatorGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface.
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Validator
}

func (p *ValidatorGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	jvspb.RegisterJVSPluginServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *ValidatorGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &GRPCClient{client: jvspb.NewJVSPluginClient(c)}, nil
}
