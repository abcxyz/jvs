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

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import com.auth0.jwk.Jwk;
import com.auth0.jwk.JwkException;
import com.auth0.jwk.JwkProvider;
import com.auth0.jwk.SigningKeyNotFoundException;
import com.auth0.jwt.interfaces.DecodedJWT;
import io.jsonwebtoken.Header;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.SignatureAlgorithm;
import io.jsonwebtoken.security.Keys;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.SecureRandom;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import javax.crypto.SecretKey;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
public class JvsClientTest {

  static KeyPair key1;
  static KeyPair key2;

  @Mock JwkProvider provider;

  @BeforeAll
  public static void setup() throws Exception {
    KeyPairGenerator keyGen = KeyPairGenerator.getInstance("EC");
    SecureRandom random = SecureRandom.getInstance("SHA1PRNG");
    keyGen.initialize(256, random);
    key1 = keyGen.generateKeyPair();
    key2 = keyGen.generateKeyPair();
  }

  @Test
  public void testValidateJWT() throws Exception {
    String keyId = "key1";

    Map<String, Object> claims = new HashMap<>();
    claims.put("id", "jwt-id");
    claims.put("role", "user");
    claims.put("created", new Date());

    String token =
        Jwts.builder()
            .setClaims(claims)
            .setHeaderParam("kid", keyId)
            .signWith(key1.getPrivate(), SignatureAlgorithm.ES256)
            .compact();

    Jwk jwk = mock(Jwk.class);
    when(jwk.getPublicKey()).thenReturn(key1.getPublic());
    when(provider.get(keyId)).thenReturn(jwk);
    JvsClient client = new JvsClient(provider, false);
    DecodedJWT returnVal = client.validateJWT(token);
    Assertions.assertEquals(claims.get("id"), returnVal.getClaims().get("id").asString());
    Assertions.assertEquals(claims.get("role"), returnVal.getClaims().get("role").asString());
    Assertions.assertEquals(
        claims.get("created"), new Date(returnVal.getClaims().get("created").asLong()));
  }

  @Test
  public void testValidateJWT_WrongKey() throws Exception {
    String keyId = "key2";

    Map<String, Object> claims = new HashMap<>();
    claims.put("id", "jwt-id");
    claims.put("role", "user");
    claims.put("created", new Date());

    String token =
        Jwts.builder()
            .setClaims(claims)
            .setHeaderParam("kid", keyId)
            .signWith(key2.getPrivate(), SignatureAlgorithm.ES256)
            .compact();

    when(provider.get(keyId))
        .thenThrow(new SigningKeyNotFoundException("", new RuntimeException()));
    JvsClient client = new JvsClient(provider, false);
    JwkException thrown =
        Assertions.assertThrows(JwkException.class, () -> client.validateJWT(token));
    assertThat(thrown.getMessage()).contains("failed to verify token");
  }

  @Test
  public void testValidateJWT_Breakglass_NotAllowed() throws Exception {
    String keyId = "key2";

    Map<String, Object> claims = new HashMap<>();
    claims.put("id", "jwt-id");
    claims.put("role", "user");
    claims.put("created", new Date());

    Justification justification = new Justification("breakglass", "issues/12345");
    claims.put("justs", List.of(justification));

    SecretKey secretKey = Keys.hmacShaKeyFor(JvsClient.BREAKGLASS_HMAC_SECRET.getBytes());
    String token =
        Jwts.builder()
            .setClaims(claims)
            .setHeaderParam("kid", keyId)
            .setHeaderParam(Header.TYPE, Header.JWT_TYPE)
            .signWith(secretKey, SignatureAlgorithm.HS256)
            .compact();

    JvsClient client = new JvsClient(provider, false);
    JwkException thrown =
        Assertions.assertThrows(JwkException.class, () -> client.validateJWT(token));
    assertThat(thrown.getMessage()).contains("breakglass is forbidden");
  }

  @Test
  public void testValidateJWT_Breakglass_Allowed() throws Exception {
    String keyId = "key2";

    Map<String, Object> claims = new HashMap<>();
    claims.put("id", "jwt-id");
    claims.put("role", "user");
    claims.put("created", new Date());

    Justification justification = new Justification("breakglass", "issues/12345");
    claims.put("justs", List.of(justification));

    SecretKey secretKey = Keys.hmacShaKeyFor(JvsClient.BREAKGLASS_HMAC_SECRET.getBytes());
    String token =
        Jwts.builder()
            .setClaims(claims)
            .setHeaderParam("kid", keyId)
            .setHeaderParam(Header.TYPE, Header.JWT_TYPE)
            .signWith(secretKey, SignatureAlgorithm.HS256)
            .compact();

    JvsClient client = new JvsClient(provider, true);
    DecodedJWT returnVal = client.validateJWT(token);
    Assertions.assertEquals(claims.get("id"), returnVal.getClaims().get("id").asString());
    Assertions.assertEquals(claims.get("role"), returnVal.getClaims().get("role").asString());
    Assertions.assertEquals(
        claims.get("created"), new Date(returnVal.getClaims().get("created").asLong()));
    Assertions.assertEquals(claims.get("justs"), List.of(justification));
  }

  @Test
  public void testValidateJWT_Breakglass_Allowed_InvalidToken() throws Exception {
    String keyId = "key2";

    Map<String, Object> claims = new HashMap<>();
    claims.put("id", "jwt-id");
    claims.put("role", "user");
    claims.put("created", new Date());

    Justification justification = new Justification("something", "else");
    claims.put("justs", List.of(justification));

    SecretKey secretKey = Keys.hmacShaKeyFor(JvsClient.BREAKGLASS_HMAC_SECRET.getBytes());
    String token =
        Jwts.builder()
            .setClaims(claims)
            .setHeaderParam("kid", keyId)
            .setHeaderParam(Header.TYPE, Header.JWT_TYPE)
            .signWith(secretKey, SignatureAlgorithm.HS256)
            .compact();

    JvsClient client = new JvsClient(provider, true);
    JwkException thrown =
        Assertions.assertThrows(JwkException.class, () -> client.validateJWT(token));
    assertThat(thrown.getMessage()).contains("failed to verify token");
  }
}
