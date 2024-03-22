// Copyright 2023 Google LLC
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

package v0

import (
	context "context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	// BreakglassHMACSecret is the HMAC key to use for creating breakglass tokens.
	// Breakglass tokens are already "unverified", so having this static secret
	// does not introduce additional risk, and breakglass is disabled by default.
	BreakglassHMACSecret = "BHzwNUbxcgpNoDfzwzt4Dr2nVXByUCWl1m8Eq2Jh26CGqu8IQ0VdiyjxnCtNahh9" //nolint:gosec

	// breakglassJustificationCategory is the category name for a breakglass
	// justification.
	breakglassJustificationCategory = "breakglass"
)

// CreateBreakglassToken creates a JWT that can be used as "breakglass" if the
// system is configured to allow breakglass tokens. The incoming [jwt.Token]
// must be built by the caller to include the standard fields. This function
// will overwrite all existing justifications, insert the breakglass
// justification, and sign JWT with an HMAC signature.
func CreateBreakglassToken(token jwt.Token, explanation string) (string, error) {
	// Set the breakglass justification.
	if err := SetJustifications(token, []*Justification{
		{
			Category: breakglassJustificationCategory,
			Value:    explanation,
		},
	}); err != nil {
		return "", fmt.Errorf("failed to set justifications on breakglass token: %w", err)
	}

	// Sign the JWT using an HMAC signature with a shared secret.
	b, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte(BreakglassHMACSecret)))
	if err != nil {
		return "", fmt.Errorf("failed to sign breakglass token: %w", err)
	}
	return string(b), nil
}

// VerifyBreakglassToken accepts an HMAC-signed JWT and verifies the signature.
// It then inspects the justifications to ensure that one of them is a
// "breakglass" justification. If successful, it returns the parsed token and
// the extracted explanation for breakglass.
func ParseBreakglassToken(ctx context.Context, tokenStr string) (jwt.Token, error) {
	message, err := jws.Parse([]byte(tokenStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token headers: %w", err)
	}

	// Check if the header is self-signed.
	found := false
	for _, signature := range message.Signatures() {
		headers := signature.ProtectedHeaders()
		if headers.Type() == "JWT" && headers.Algorithm() == jwa.HS256 {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}

	token, err := jwt.Parse([]byte(tokenStr),
		jwt.WithContext(ctx),
		jwt.WithKey(jwa.HS256, []byte(BreakglassHMACSecret)),
		jwt.WithAcceptableSkew(5*time.Second),
		WithTypedJustifications(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse breakglass jwt: %w", err)
	}

	justifications, err := GetJustifications(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get justifications from token: %w", err)
	}

	explanation := ""
	for _, justification := range justifications {
		if justification.GetCategory() == breakglassJustificationCategory {
			explanation = justification.GetValue()
			break
		}
	}
	if explanation == "" {
		return nil, fmt.Errorf("failed to find breakglass justification token")
	}

	return token, nil
}
