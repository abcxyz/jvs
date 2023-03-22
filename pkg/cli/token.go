// Copyright 2022 Google LLC
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

package cli

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/durationpb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/cli"
)

var _ cli.Command = (*TokenCommand)(nil)

type TokenCommand struct {
	cli.BaseCommand

	flagAudiences   []string
	flagAuthToken   string
	flagBreakglass  bool
	flagExplanation string
	flagSubject     string
	flagTTL         time.Duration

	// flag Server is the server address.
	flagServer string

	// flagInsecure controls whether to use insecure grpc transport credentials.
	flagInsecure bool

	// flagNowUnix is a hidden flag that's used to override the current timestamp
	// for testing.
	flagNowUnix int64
}

func (c *TokenCommand) Desc() string {
	return `Generate a justification token`
}

func (c *TokenCommand) Help() string {
	return strings.Trim(`
Generate a new justification token from the given JVS. The output will be the
token, or any errors that occurred.

EXAMPLES

    # Generate a token with a 30min ttl
    jvsctl token \
      -explanation "issues/12345" \
      -ttl "30m"

    # Generate a token with custom audiences
    jvsctl token \
      -explanation "access production" \
      -audiences "my.service.dev"

    # Generate a breakglass token
    jvsctl token \
      -explanation "everything is broken" \
      -breakglass

`+c.Flags().Help(), "\n")
}

func (c *TokenCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "audience",
		Target:  &c.flagAudiences,
		Aliases: []string{"aud"},
		Example: "org.corp.example,net.corp.example",
		EnvVar:  "JVSCTL_TOKEN_AUDIENCES",
		Usage: `The list of audiences to include in the generated ` +
			`justification token.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "auth-token",
		Target:  &c.flagAuthToken,
		Example: "ya29.c...",
		EnvVar:  "JVSCTL_AUTH_TOKEN",
		Usage:   `An OIDC token to use for authentication.`,
	})

	f.BoolVar(&cli.BoolVar{
		Name:    "breakglass",
		Target:  &c.flagBreakglass,
		Default: false,
		Usage:   `Make a breakglass token.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "explanation",
		Target:  &c.flagExplanation,
		Aliases: []string{"e"},
		Example: "Debugging ticket #123",
		Usage:   `A reason for the action.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "subject",
		Target:  &c.flagSubject,
		Example: "you@example.com",
		EnvVar:  "JVSCTL_TOKEN_SUBJECT",
		Usage:   `The principal that will be using the token.`,
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "ttl",
		Target:  &c.flagTTL,
		Example: "5m",
		Default: 15 * time.Minute,
		EnvVar:  "JVSCTL_TOKEN_TTL",
		Usage:   `The token lifetime, as a duration.`,
	})

	f.Int64Var(&cli.Int64Var{
		Name:    "now",
		Target:  &c.flagNowUnix,
		Default: time.Now().Unix(),
		Hidden:  true,
		Usage:   `Current timestamp, in unix seconds.`,
	})

	// Server flags
	f = set.NewSection("SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "server",
		Target:  &c.flagServer,
		Example: "https://jvs.example.com:8080",
		Default: "http://localhost:8080",
		EnvVar:  "JVSCTL_SERVER_ADDRESS",
		Usage:   `JVS server address including the protocol, address, and port.`,
	})

	f.BoolVar(&cli.BoolVar{
		Name:    "insecure",
		Target:  &c.flagInsecure,
		Default: false,
		Usage:   "Use an insecure grpc connection.",
	})

	return set
}

func (c *TokenCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if c.flagExplanation == "" {
		return fmt.Errorf("explanation is required")
	}

	// Explicitly set this here because, if it's set as a default, there's no way
	// to remove it.
	if len(c.flagAudiences) == 0 {
		c.flagAudiences = []string{justification.DefaultAudience}
	}

	// breakglass won't require JVS server. Handle that first.
	if c.flagBreakglass {
		fmt.Fprintln(c.Stderr(), "WARNING: In breakglass mode, the justification token is not signed.")
		tok, err := c.breakglassToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate breakglass token: %w", err)
		}
		fmt.Fprintln(c.Stdout(), tok)
		return nil
	}

	dialOpts, err := dialOptions(c.flagInsecure)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(c.flagServer, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to JVS service: %w", err)
	}
	jvsclient := jvspb.NewJVSServiceClient(conn)

	callOpts, err := callOptions(ctx, c.flagAuthToken)
	if err != nil {
		return err
	}

	req := &jvspb.CreateJustificationRequest{
		Subject: c.flagSubject,
		Justifications: []*jvspb.Justification{{
			Category: "explanation",
			Value:    c.flagExplanation,
		}},
		Ttl: durationpb.New(c.flagTTL),
	}
	resp, err := jvsclient.CreateJustification(ctx, req, callOpts...)
	if err != nil {
		return fmt.Errorf("failed to create justification: %w", err)
	}

	fmt.Fprintln(c.Stdout(), resp.Token)
	return nil
}

func dialOptions(insecure bool) ([]grpc.DialOption, error) {
	if insecure {
		return []grpc.DialOption{
			grpc.WithTransportCredentials(grpcinsecure.NewCredentials()),
		}, nil
	}

	// The default.
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system cert pool: %w", err)
	}
	//nolint:gosec // We need to support TLS 1.2 for now (G402).
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	return []grpc.DialOption{
		grpc.WithTransportCredentials(cred),
	}, nil
}

func callOptions(ctx context.Context, authToken string) ([]grpc.CallOption, error) {
	if authToken == "" {
		return nil, nil
	}

	token := &oauth2.Token{
		AccessToken: authToken,
	}

	rpcCreds := oauth.TokenSource{
		TokenSource: oauth2.StaticTokenSource(token),
	}

	return []grpc.CallOption{
		grpc.PerRPCCredentials(rpcCreds),
	}, nil
}

// breakglassToken creates a new breakglass token from the CLI flags. See
// [jvspb.CreateBreakglassToken] for more information.
func (c *TokenCommand) breakglassToken(ctx context.Context) (string, error) {
	now := time.Unix(c.flagNowUnix, 0)
	id := uuid.New().String()
	exp := now.Add(c.flagTTL)

	token, err := jwt.NewBuilder().
		Audience(c.flagAudiences).
		Expiration(exp).
		IssuedAt(now).
		Issuer(Issuer).
		JwtID(id).
		NotBefore(now).
		Subject(c.flagSubject).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build breakglass token: %w", err)
	}

	str, err := jvspb.CreateBreakglassToken(token, c.flagExplanation)
	if err != nil {
		return "", fmt.Errorf("failed to create breakglass token: %w", err)
	}
	return str, nil
}
