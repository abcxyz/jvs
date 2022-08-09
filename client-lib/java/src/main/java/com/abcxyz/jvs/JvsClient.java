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

import com.auth0.jwk.Jwk;
import com.auth0.jwk.JwkException;
import com.auth0.jwk.JwkProvider;
import com.auth0.jwk.SigningKeyNotFoundException;
import com.auth0.jwt.JWT;
import com.auth0.jwt.algorithms.Algorithm;
import com.auth0.jwt.interfaces.DecodedJWT;
import java.security.interfaces.ECPublicKey;
import java.util.List;
import java.util.Map;
import lombok.AccessLevel;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/** A Client for use when validating JVS tokens. Can be build using JVSClientBuilder. */
@RequiredArgsConstructor(access = AccessLevel.PACKAGE)
@Slf4j
public class JvsClient {
  // This postfix is added by the cli tool when creating breakglass tokens.
  private static final String UNSIGNED_POSTFIX = ".NOT_SIGNED";

  // this category is added by the cli tool when creating breakglass tokens.
  private static final String BREAKGLASS_CATEGORY = "breakglass";

  private final JwkProvider provider;
  private final boolean allowBreakglass;

  public DecodedJWT validateJWT(String jwtString) throws JwkException {
    DecodedJWT jwt = JWT.decode(jwtString);

    // Handle unsigned tokens.
    if (jwtString.endsWith(UNSIGNED_POSTFIX)) {
      if (unsignedTokenValidAndAllowed(jwt)) {
        return jwt;
      } else {
        throw new JwkException("Token unsigned and could not be validated.");
      }
    }

    try {
      Jwk jwk = provider.get(jwt.getKeyId());
      Algorithm algorithm = Algorithm.ECDSA256((ECPublicKey) jwk.getPublicKey(), null);
      algorithm.verify(jwt);
    } catch (SigningKeyNotFoundException e) {
      log.info("No public key found with id: {}", jwt.getKeyId());
      throw new JwkException("Public key not found", e);
    }

    return jwt;
  }

  // Check that the unsigned token is valid and that we allow break-glass tokens.
  boolean unsignedTokenValidAndAllowed(DecodedJWT jwt) {
    if (!allowBreakglass) {
      log.info(""breakglass is forbidden, denying"");
      return false;
    }
    List<Map> justifications = jwt.getClaim("justs").asList(Map.class);
    for (Map<String, String> justification : justifications) {
      if (justification.getOrDefault("category", "").equals(BREAKGLASS_CATEGORY)) {
        return true;
      }
    }
    log.info("justification category is not breakglass, denying.");
    return false;
  }
}
