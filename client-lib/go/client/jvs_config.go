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

package client

import (
	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// JVSConfig is the config. This is an alias for jvspb.
//
// Deprecated: Use [jvspb.Config] directly instead.
type JVSConfig = jvspb.Config

// LoadJVSConfig calls the necessary methods to load in config using the
// OsLookuper which finds env variables specified on the host.
//
// Deprecated: Use [jvspb.LoadConfig] directly instead.
var LoadJVSConfig = jvspb.LoadConfig
