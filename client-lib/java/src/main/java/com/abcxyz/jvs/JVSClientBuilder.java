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

import com.auth0.jwk.JwkProvider;
import com.auth0.jwk.JwkProviderBuilder;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.google.api.client.util.Strings;
import java.io.IOException;
import java.io.InputStream;
import java.time.Duration;
import java.time.format.DateTimeParseException;
import lombok.AccessLevel;
import lombok.Getter;

/**
 * JVSClientBuilder can be used to build JVS clients with a cached JwkProvider. It has values that
 * need to be set, and those may be specified in a yaml file through loadConfigFromFile, or
 * environment variables. Set environment variables are automatically loaded when build() is called.
 * If a yaml and environment variable are both set, the env variable is used.
 *
 * env: JVS_VERSION yaml: version. Specifies the version of the configuration.
 * env: JVS_CACHE_TIMEOUT yaml: cache_timeout. Specifies how long until cached public keys are
 * invalidated.
 * env: JVS_ENDPOINT yaml: endpoint. Specifies the url for retrieving public keys.
 */
public class JVSClientBuilder {

  static final String ENDPOINT_ENV_KEY = "JVS_ENDPOINT";
  static final String CACHE_TIMEOUT_ENV_KEY = "JVS_CACHE_TIMEOUT";
  static final String VERSION_ENV_KEY = "JVS_VERSION";

  @Getter(AccessLevel.PACKAGE)
  JVSConfiguration configuration;

  public JVSClientBuilder loadConfigFromFile(String fileName) throws IOException {
    try (InputStream input = getClass().getClassLoader().getResourceAsStream(fileName)) {
      ObjectMapper mapper = new ObjectMapper(new YAMLFactory());
      mapper.registerModule(new JavaTimeModule());
      configuration = mapper.readValue(input, JVSConfiguration.class);
      return this;
    }
  }

  void updateConfigFromEnvironmentVars() throws DateTimeParseException {
    if (configuration == null) {
      configuration = new JVSConfiguration();
    }

    String endpointEnv = getFromEnvironmentVars(ENDPOINT_ENV_KEY);
    if (!Strings.isNullOrEmpty(endpointEnv)) {
      configuration.setJvsEndpoint(endpointEnv);
    }

    String versionEnv = getFromEnvironmentVars(VERSION_ENV_KEY);
    if (!Strings.isNullOrEmpty(versionEnv)) {
      configuration.setVersion(versionEnv);
    }

    String timeoutEnv = getFromEnvironmentVars(CACHE_TIMEOUT_ENV_KEY);
    if (!Strings.isNullOrEmpty(timeoutEnv)) {
      configuration.setCacheTimeout(Duration.parse(timeoutEnv));
    }
  }

  String getFromEnvironmentVars(String key) {
    return System.getenv().getOrDefault(key, null);
  }

  public JVSClient build() {
    // Load env vars and validate config
    updateConfigFromEnvironmentVars();
    configuration.validate();

    JwkProvider provider = new JwkProviderBuilder(configuration.getJvsEndpoint())
        .cached(10, configuration.getCacheTimeout())
        // TODO: by default, the rate limiter allows 10 reqs per minute.
        // https://github.com/auth0/jwks-rsa-java/blob/master/src/main/java/com/auth0/jwk/JwkProviderBuilder.java#L43
        // we can consider if a different rate limit makes more sense for us.
        .rateLimited(true)
        .build();

    return new JVSClient(provider);
  }
}
