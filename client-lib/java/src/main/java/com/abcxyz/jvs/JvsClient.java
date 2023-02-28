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
import com.auth0.jwt.exceptions.SignatureVerificationException;
import com.auth0.jwt.interfaces.DecodedJWT;
import io.jsonwebtoken.Header;
import io.jsonwebtoken.SignatureAlgorithm;
import java.security.interfaces.ECPublicKey;
import java.util.List;
import lombok.AccessLevel;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/** A Client for use when validating JVS tokens. Can be build using JVSClientBuilder. */
@RequiredArgsConstructor(access = AccessLevel.PACKAGE)
@Slf4j
public class JvsClient {
  // This is the HMAC key to use for creating breakglass tokens. Breakglass
  // tokens are already "unverified", so having this static secret does not
  // introduce additional risk, and breakglass is disabled by default.
  protected static final String BREAKGLASS_HMAC_SECRET =
      "BHzwNUbxcgpNoDfzwzt4Dr2nVXByUCWl1m8Eq2Jh26CGqu8IQ0VdiyjxnCtNahh9";

  // This is the justification category set for breakglass tokens.
  private static final String BREAKGLASS_JUSTIFICATION_CATEGORY = "breakglass";

  private final JwkProvider provider;
  private final boolean allowBreakglass;

  /**
   * This parses the given token as a breakglass token. If the token's signature is invalid, it
   * throws an exception. If the token does not contain the breakglass justification an exception is
   * thrown.
   */
  private void validateBreakglassToken(DecodedJWT token) throws JwkException {
    try {
      Algorithm alg = Algorithm.HMAC256(BREAKGLASS_HMAC_SECRET);
      alg.verify(token);
    } catch (SignatureVerificationException e) {
      log.error("failed to verify breakglass token: {}", e);
      throw new JwkException("failed to parse breakglass jwt", e);
    }

    List<Justification> justifications = token.getClaim("justs").asList(Justification.class);
    for (Justification justification : justifications) {
      if (BREAKGLASS_JUSTIFICATION_CATEGORY.equals(justification.getCategory())) {
        return;
      }
    }

    throw new JwkException("token is breakglass, but is missing breakglass justification");
  }

  /**
   * This takes a jwt string, converts it to a JWT, and validates the signature against the JWKs
   * endpoint. It also handles breakglass, if breakglass is enabled.
   */
  public DecodedJWT validateJWT(String tokenStr, String expectedSubject) throws JwkException {
    DecodedJWT token = JWT.decode(tokenStr);

    // Handle breakglass tokens
    if (Header.JWT_TYPE.equals(token.getType())
        && SignatureAlgorithm.HS256.getValue().equals(token.getAlgorithm())) {
      log.debug("token is breakglass");
      if (!allowBreakglass) {
        throw new JwkException("breakglass is forbidden, denying");
      }
      validateBreakglassToken(token);
      return token;
    }

    // Handle regular token processing.
    try {
      Jwk jwk = provider.get(token.getKeyId());
      if (jwk == null) {
        throw new SigningKeyNotFoundException("daf", null);
      }
      Algorithm algorithm = Algorithm.ECDSA256((ECPublicKey) jwk.getPublicKey(), null);
      algorithm.verify(token);

      if (!expectedSubject.equals(token.getSubject()) && !expectedSubject.isBlank()) {
        String msg =
            String.format(
                "subject %s does not match expected subject %s",
                token.getSubject(), expectedSubject);
        log.error(msg);
        throw new JwkException(msg);
      }
    } catch (SigningKeyNotFoundException e) {
      log.error("failed to find public key {}", token.getKeyId());
      throw new JwkException("failed to verify token", e);
    }

    return token;
  }
}
