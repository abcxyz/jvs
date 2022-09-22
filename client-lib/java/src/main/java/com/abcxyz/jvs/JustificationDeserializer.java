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

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.ObjectCodec;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.deser.std.StdDeserializer;
import java.io.IOException;

public class JustificationDeserializer extends StdDeserializer<Justification> {
  public JustificationDeserializer() {
    this(null);
  }

  public JustificationDeserializer(Class<?> vc) {
    super(vc);
  }

  @Override
  public Justification deserialize(JsonParser jsonParser, DeserializationContext context)
      throws IOException, JsonProcessingException {

    ObjectCodec codec = jsonParser.getCodec();
    if (codec == null) {
      return null;
    }

    JsonNode node = codec.readTree(jsonParser);
    if (node == null) {
      return null;
    }

    String category = null;
    JsonNode categoryNode = node.get("category");
    if (categoryNode != null) {
      category = categoryNode.asText();
    }

    String value = null;
    JsonNode valueNode = node.get("value");
    if (valueNode != null) {
      value = valueNode.asText();
    }

    return new Justification(category, value);
  }
}
