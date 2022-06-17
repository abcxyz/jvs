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
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/durationpb"

	jvsapis "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/idtoken"
)

var (
	tokenExplanation string
	breakglass       bool
	ttl              time.Duration
)

var tokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "To generate a justification token",
	Example: `token --explanation "issues/12345" -ttl 30m`,
	RunE:    runTokenCmd,
}

func runTokenCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	dialOpt, err := dialOpt()
	if err != nil {
		return err
	}
	callOpt, err := callOpt(ctx)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(cfg.Server, dialOpt)
	if err != nil {
		return fmt.Errorf("failed to connect to JVS service: %w", err)
	}
	jvsclient := jvsapis.NewJVSServiceClient(conn)

	req := &jvsapis.CreateJustificationRequest{
		Justifications: []*jvsapis.Justification{{
			Category: "explanation",
			Value:    tokenExplanation,
		}},
		Ttl: durationpb.New(ttl),
	}
	resp, err := jvsclient.CreateJustification(ctx, req, callOpt)
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write([]byte(resp.Token))
	return err
}

func init() {
	tokenCmd.Flags().StringVarP(&tokenExplanation, "explanation", "e", "", "The explanation for the action")
	tokenCmd.MarkFlagRequired("explanation") //nolint // not expect err
	tokenCmd.Flags().BoolVar(&breakglass, "breakglass", false, "Whether it will be a breakglass action")
	tokenCmd.Flags().DurationVar(&ttl, "ttl", time.Hour, "The token time-to-live duration")
}

func dialOpt() (grpc.DialOption, error) {
	if cfg.Authentication.Insecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
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
	return grpc.WithTransportCredentials(cred), nil
}

func callOpt(ctx context.Context) (grpc.CallOption, error) {
	if cfg.Authentication.Insecure {
		return nil, nil
	}

	ts, err := idtoken.FromDefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	return grpc.PerRPCCredentials(oauth.NewOauthAccess(token)), nil
}
