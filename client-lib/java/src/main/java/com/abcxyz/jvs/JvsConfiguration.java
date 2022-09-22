/*
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.abcxyz.jvs;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.google.api.client.util.Strings;
import java.time.Duration;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Configuration for use with the JVSClientBuilder. See JVSClientBuilder for detail on the values.
 */
@Data
@NoArgsConstructor
public class JvsConfiguration {

  private static final String EXPECTED_VERSION = "1";

  @JsonProperty(value = "version")
  private String version = EXPECTED_VERSION;

  @JsonProperty("jwks_endpoint")
  private String jwksEndpoint = "http://localhost:8080/.well-known/jwks";

  @JsonProperty("cache_timeout")
  private Duration cacheTimeout = Duration.ofMinutes(5);

  @JsonProperty("allow_breakglass")
  private boolean breakglassAllowed = false;

  public void validate() throws IllegalArgumentException {
    if (!version.equals(EXPECTED_VERSION)) {
      throw new IllegalArgumentException(
          String.format("Wrong version. Got %s, but wanted %s", version, EXPECTED_VERSION));
    }

    if (Strings.isNullOrEmpty(jwksEndpoint)) {
      throw new IllegalArgumentException("JWKs endpoint was not specified.");
    }

    if (cacheTimeout == null || cacheTimeout.isNegative() || cacheTimeout.isZero()) {
      throw new IllegalArgumentException(
          String.format(
              "Cache timeout is invalid. Must be a positive non-zero duration, but was set to %s",
              cacheTimeout));
    }
  }
}
