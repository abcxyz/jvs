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

import static org.mockito.Mockito.spy;
import static org.mockito.Mockito.when;

import java.time.Duration;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

public class JVSClientBuilderTest {

  @Test
  public void testLoadConfigurationFromFile() {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("all_specified.yml");

    JVSConfiguration expectedConfig = new JVSConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT5M"));

    JVSConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    loadedConfig.validate(); // shouldn't throw exception.
  }

  @Test
  public void testLoadConfigurationFromFile_Invalid() {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("invalid.yml");

    JVSConfiguration expectedConfig = new JVSConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT-5M"));

    JVSConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);

    // invalid values, expect exception.
    Assertions.assertThrows(IllegalArgumentException.class, () -> loadedConfig.validate());
  }

  @Test
  public void testLoadConfigurationFromFile_MissingValues_EnvVars() {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    builder.loadConfigFromFile("missing_values.yml");

    JVSConfiguration expectedConfig = new JVSConfiguration();
    expectedConfig.setVersion("1");

    JVSConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    // values missing, expect exception.
    Assertions.assertThrows(IllegalArgumentException.class, () -> loadedConfig.validate());

    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn(
        "google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn(
        "PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JVSConfiguration loadedConfig2 = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig2);
    loadedConfig2.validate(); // shouldn't throw exception.
  }

  @Test
  public void testLoadConfigurationFromFile_JustEnvVars() {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    JVSConfiguration expectedConfig = new JVSConfiguration();

    when(builder.getFromEnvironmentVars(JVSClientBuilder.VERSION_ENV_KEY)).thenReturn(
        "1");
    expectedConfig.setVersion("1");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn(
        "google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn(
        "PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JVSConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    loadedConfig.validate(); // shouldn't throw exception.
  }

  @Test
  public void testLoadConfigurationFromFile_EnvVarsOverride() {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    builder.loadConfigFromFile("all_specified.yml");

    JVSConfiguration expectedConfig = new JVSConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT5M"));

    JVSConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    loadedConfig.validate(); // shouldn't throw exception.

    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn(
        "google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn(
        "PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JVSConfiguration loadedConfig2 = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig2);
    loadedConfig2.validate(); // shouldn't throw exception.
  }

  @Test
  public void testBuild() {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("all_specified.yml");
    JVSClient client = builder.build();
    Assertions.assertNotNull(client);
  }

  @Test()
  public void testBuild_Fail() {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("missing_values.yml");
    Assertions.assertThrows(IllegalArgumentException.class, () -> builder.build());
  }
}
