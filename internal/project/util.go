// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package project defines global project helpers.
package project

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// devMode indicates whether the project is running in development mode.
var devMode, _ = strconv.ParseBool(os.Getenv("DEV_MODE"))

var _, self, _, _ = runtime.Caller(0)

// Root returns the filepath to the root of this project.
func Root(more ...string) string {
	root := []string{filepath.Dir(self), "..", ".."}
	root = append(root, more...)
	return filepath.Join(root...)
}

// DevMode indicates whether the project is running in development mode.
func DevMode() bool {
	return devMode
}
