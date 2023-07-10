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
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	grpc "google.golang.org/grpc"
)

const (
	handshakeCookieKey   = "JVS_PLUGIN"
	handshakeCookieValue = "cc400ef1c6e74ee20be491c6013ae2120fb04c11703d05fbbf18dbb2e5e0"
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

// The interface we are exposing as a plugin.
type Validator interface {
	Validate(context.Context, *ValidateJustificationRequest) (*ValidateJustificationResponse, error)
}

// This is the implementation of [plugin.GRPCPlugin] so we can serve/consume this.
//
// [plugin.GRPCPlugin]: https://github.com/hashicorp/go-plugin/blob/a88a423a8813d0b26c8e3219f71b0f30447b5d2e/plugin.go#L36
type ValidatorPlugin struct {
	// GRPCPlugin must still implement the Plugin interface.
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Validator
}

// Due to [type check], ValidatorPlugin need to implement the following interface.
//
// [type check]: https://github.com/hashicorp/go-plugin/blob/a88a423a8813d0b26c8e3219f71b0f30447b5d2e/server.go#L191
func (p *ValidatorPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterJVSPluginServer(s, &PluginServer{Impl: p.Impl})
	return nil
}

func (p *ValidatorPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &PluginClient{client: NewJVSPluginClient(c)}, nil
}

// PluginClient is an implementation of Validator that talks over RPC.
type PluginClient struct {
	client JVSPluginClient
}

func (m *PluginClient) Validate(ctx context.Context, req *ValidateJustificationRequest) (*ValidateJustificationResponse, error) {
	resp, err := m.client.Validate(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to validate justification: %w", err)
	}
	return resp, nil
}

// Here is the gRPC server that PluginClient talks to.
type PluginServer struct {
	JVSPluginServer
	// This is the real implementation
	Impl Validator
}

func (m *PluginServer) Validate(ctx context.Context, req *ValidateJustificationRequest) (*ValidateJustificationResponse, error) {
	resp, err := m.Impl.Validate(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to validate justification: %w", err)
	}
	return resp, nil
}
