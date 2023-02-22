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
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/durationpb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/idtoken"
	"github.com/abcxyz/jvs/pkg/justification"
)

// tokenCmdOptions holds all the inputs and flags for the token subcommand.
type tokenCmdOptions struct {
	config *config.CLIConfig

	audiences    []string
	explanation  string
	breakglass   bool
	ttl          time.Duration
	issTimeUnix  int64
	disableAuthn bool
}

// newTokenCmd creates a new subcommand for issuing tokens.
func newTokenCmd(cfg *config.CLIConfig) *cobra.Command {
	opts := &tokenCmdOptions{
		config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Generate a justification token",
		Long: strings.Trim(`
Generate a new justification token from the given JVS. The output will be the
token, or any errors that occurred.

For example:

    # Generate a token with a 30min ttl
    jvsctl token --explanation "issues/12345" --ttl 30m

    # Generate a token with custom audiences
    jvsctl token --explanation "access production" --audiences "my.service.dev"

    # Generate a breakglass token
    jvsctl token --explanation "everything is broken" --breakglass
`, "\n"),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTokenCmd(cmd, opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.explanation, "explanation", "e", "",
		"The explanation for the action")
	cmd.MarkFlagRequired("explanation") //nolint // not expect err
	flags.StringSliceVar(&opts.audiences, "audiences", []string{justification.DefaultAudience},
		"The list of audiences for the token")
	flags.BoolVar(&opts.breakglass, "breakglass", false,
		"Whether it will be a breakglass action")
	flags.DurationVar(&opts.ttl, "ttl", 15*time.Minute,
		"The token time-to-live duration")
	flags.Int64Var(&opts.issTimeUnix, "iat", time.Now().Unix(),
		"A hidden flag to specify token issue time")
	flags.MarkHidden("iat") //nolint // not expect err
	flags.BoolVar(&opts.disableAuthn, "disable-authn", false,
		"A hidden flag to disable authentication")
	flags.MarkHidden("disable-authn") //nolint // not expect err

	return cmd
}

func runTokenCmd(cmd *cobra.Command, opts *tokenCmdOptions, args []string) error {
	ctx := context.Background()
	out := cmd.OutOrStdout()

	// breakglass won't require JVS server. Handle that first.
	if opts.breakglass {
		fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: In breakglass mode, the justification token is not signed.")
		tok, err := breakglassToken(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to generate breakglass token: %w", err)
		}
		fmt.Fprintln(out, tok)
		return nil
	}

	dialOpts, err := dialOpts(opts.config.Insecure)
	if err != nil {
		return err
	}
	callOpts, err := callOpts(ctx, opts.disableAuthn)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(opts.config.Server, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to JVS service: %w", err)
	}
	jvsclient := jvspb.NewJVSServiceClient(conn)

	req := &jvspb.CreateJustificationRequest{
		Justifications: []*jvspb.Justification{{
			Category: "explanation",
			Value:    opts.explanation,
		}},
		Ttl: durationpb.New(opts.ttl),
	}
	resp, err := jvsclient.CreateJustification(ctx, req, callOpts...)
	if err != nil {
		return fmt.Errorf("failed to create justification: %w", err)
	}

	fmt.Fprintln(out, resp.Token)
	return nil
}

func dialOpts(insecure bool) ([]grpc.DialOption, error) {
	if insecure {
		return []grpc.DialOption{grpc.WithTransportCredentials(grpcinsecure.NewCredentials())}, nil
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
	return []grpc.DialOption{grpc.WithTransportCredentials(cred)}, nil
}

func callOpts(ctx context.Context, disableAuthn bool) ([]grpc.CallOption, error) {
	if disableAuthn {
		return nil, nil
	}

	ts, err := idtoken.FromDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default credentials: %w", err)
	}

	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token from default credentials: %w", err)
	}

	rpcCreds := oauth.TokenSource{
		TokenSource: oauth2.StaticTokenSource(token),
	}

	return []grpc.CallOption{grpc.PerRPCCredentials(rpcCreds)}, nil
}

// breakglassToken creates a new breakglass token from the CLI flags. See
// [jvspb.CreateBreakglassToken] for more information.
func breakglassToken(ctx context.Context, opts *tokenCmdOptions) (string, error) {
	now := time.Unix(opts.issTimeUnix, 0)
	id := uuid.New().String()
	exp := now.Add(opts.ttl)

	token, err := jwt.NewBuilder().
		Audience(opts.audiences).
		Expiration(exp).
		IssuedAt(now).
		Issuer(Issuer).
		JwtID(id).
		NotBefore(now).
		Subject(Subject).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build breakglass token: %w", err)
	}

	str, err := jvspb.CreateBreakglassToken(token, opts.explanation)
	if err != nil {
		return "", fmt.Errorf("failed to create breakglass token: %w", err)
	}
	return str, nil
}
