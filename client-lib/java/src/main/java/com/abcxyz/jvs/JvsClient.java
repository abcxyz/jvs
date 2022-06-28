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
import lombok.AccessLevel;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/** A Client for use when validating JVS tokens. Can be build using JVSClientBuilder. */
@RequiredArgsConstructor(access = AccessLevel.PACKAGE)
@Slf4j
public class JvsClient {

  private final JwkProvider provider;

  public DecodedJWT validateJWT(String jwtString) throws JwkException {
    DecodedJWT jwt = JWT.decode(jwtString);
    Jwk jwk;
    try {
      jwk = provider.get(jwt.getKeyId());
    } catch (SigningKeyNotFoundException e) {
      log.info("No public key found with id: {}", jwt.getKeyId());
      throw new JwkException("Public key not found", e);
    }
    Algorithm algorithm = Algorithm.ECDSA256((ECPublicKey) jwk.getPublicKey(), null);
    algorithm.verify(jwt);
    return jwt;
  }
}
