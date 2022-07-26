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

// Package util provides utilities that are intended to enable easier
// and more concise writing of source code.
package util

import (
	"io"

	"go.uber.org/zap"
)

// GracefulClose calls Close() and logs the error if there is.
func GracefulClose(logger *zap.SugaredLogger, c io.Closer) {
	if err := c.Close(); err != nil {
		logger.Errorf("failed to close: %v", err)
	}
}
