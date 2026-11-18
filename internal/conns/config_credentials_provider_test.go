// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package conns_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hashicorp/aws-sdk-go-base/v2/servicemocks"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

type countingCredentialsProvider struct {
	calls int
}

func (p *countingCredentialsProvider) Retrieve(context.Context) (aws.Credentials, error) {
	p.calls++
	return aws.Credentials{
		AccessKeyID:     "InjectedAccessKey",
		SecretAccessKey: "InjectedSecretKey",
		SessionToken:    "InjectedSessionToken",
		Source:          "injected",
	}, nil
}

func TestConfigureProvider_credentialsProvider(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		injected          bool
		expectedAccessKey string
	}{
		"static credentials only": {
			expectedAccessKey: "StaticAccessKey",
		},
		"injected credentials provider overrides static credentials": {
			injected:          true,
			expectedAccessKey: "InjectedAccessKey",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			injected := &countingCredentialsProvider{}
			config := conns.Config{
				AccessKey:               "StaticAccessKey",
				SecretKey:               servicemocks.MockStaticSecretKey,
				Region:                  "us-west-2",
				SkipCredsValidation:     true,
				SkipRequestingAccountId: true,
			}
			if tc.injected {
				config.CredentialsProvider = injected
			}

			client, diags := config.ConfigureProvider(ctx, &conns.AWSClient{})
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}

			cfg := client.AwsConfig(ctx)
			if tc.injected && cfg.Credentials != injected {
				t.Fatalf("expected the injected credentials provider to be used, got %T", cfg.Credentials)
			}
			creds, err := cfg.Credentials.Retrieve(ctx)
			if err != nil {
				t.Fatalf("unexpected error retrieving credentials: %s", err)
			}
			if creds.AccessKeyID != tc.expectedAccessKey {
				t.Errorf("expected access key %q, got %q", tc.expectedAccessKey, creds.AccessKeyID)
			}
			if !tc.injected && injected.calls != 0 {
				t.Errorf("expected the injected credentials provider not to be called, got %d calls", injected.calls)
			}
		})
	}
}
