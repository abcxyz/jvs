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

public class JvsClientBuilderTest {

  @Test
  public void testLoadConfigurationFromFile() throws Exception {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("all_specified.yml");

    JvsConfiguration expectedConfig = new JvsConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT5M"));

    JvsConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    Assertions.assertDoesNotThrow(() -> loadedConfig.validate());
  }

  @Test
  public void testLoadConfigurationFromFile_Invalid() throws Exception {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("invalid.yml");

    JvsConfiguration expectedConfig = new JvsConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT-5M"));

    JvsConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);

    // invalid values, expect exception.
    IllegalArgumentException thrown = Assertions.assertThrows(IllegalArgumentException.class, () -> loadedConfig.validate());
    Assertions.assertTrue(thrown.getMessage().contains("Cache timeout is invalid. Must be a positive non-zero duration"));
  }

  @Test
  public void testLoadConfigurationFromFile_MissingValues_EnvVars() throws Exception {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    builder.loadConfigFromFile("missing_values.yml");

    JvsConfiguration expectedConfig = new JvsConfiguration();
    expectedConfig.setVersion("1");

    JvsConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    // values missing, expect exception.
    IllegalArgumentException thrown = Assertions.assertThrows(IllegalArgumentException.class, () -> loadedConfig.validate());
    Assertions.assertTrue(thrown.getMessage().contains("JVS endpoint was not specified"));

    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn(
        "google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn(
        "PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JvsConfiguration loadedConfig2 = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig2);
    Assertions.assertDoesNotThrow(() -> loadedConfig2.validate());
  }

  @Test
  public void testLoadConfigurationFromFile_JustEnvVars() throws Exception {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    JvsConfiguration expectedConfig = new JvsConfiguration();

    when(builder.getFromEnvironmentVars(JVSClientBuilder.VERSION_ENV_KEY)).thenReturn("1");
    expectedConfig.setVersion("1");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn("google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn("PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JvsConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    Assertions.assertDoesNotThrow(() -> loadedConfig.validate());
  }

  @Test
  public void testLoadConfigurationFromFile_EnvVarsOverride() throws Exception {
    JVSClientBuilder b = new JVSClientBuilder();
    JVSClientBuilder builder = spy(b);

    builder.loadConfigFromFile("all_specified.yml");

    JvsConfiguration expectedConfig = new JvsConfiguration();
    expectedConfig.setVersion("1");
    expectedConfig.setJvsEndpoint("example.com");
    expectedConfig.setCacheTimeout(Duration.parse("PT5M"));

    JvsConfiguration loadedConfig = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig);
    Assertions.assertDoesNotThrow(() -> loadedConfig.validate());

    when(builder.getFromEnvironmentVars(JVSClientBuilder.ENDPOINT_ENV_KEY)).thenReturn(
        "google.com");
    expectedConfig.setJvsEndpoint("google.com");
    when(builder.getFromEnvironmentVars(JVSClientBuilder.CACHE_TIMEOUT_ENV_KEY)).thenReturn(
        "PT10M");
    expectedConfig.setCacheTimeout(Duration.parse("PT10M"));

    builder.updateConfigFromEnvironmentVars();

    JvsConfiguration loadedConfig2 = builder.getConfiguration();
    Assertions.assertEquals(expectedConfig, loadedConfig2);
    Assertions.assertDoesNotThrow(() -> loadedConfig2.validate());
  }

  @Test
  public void testBuild() throws Exception {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("all_specified.yml");
    JvsClient client = builder.build();
    Assertions.assertNotNull(client);
  }

  @Test()
  public void testBuild_Fail() throws Exception {
    JVSClientBuilder builder = new JVSClientBuilder();
    builder.loadConfigFromFile("missing_values.yml");
    IllegalArgumentException thrown = Assertions.assertThrows(IllegalArgumentException.class, () -> builder.build());
    Assertions.assertTrue(thrown.getMessage().contains("JVS endpoint was not specified"));
  }
}
