package xpprovider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/provider"
)

type AWSConfig conns.Config
type AWSClient conns.AWSClient

func GetProviderSchema(ctx context.Context) (*schema.Provider, error) {
	return provider.New(ctx)
}

func (ac *AWSConfig) GetClient(ctx context.Context, client *AWSClient) (*conns.AWSClient, diag.Diagnostics) {
	return (*conns.Config)(ac).ConfigureProvider(ctx, (*conns.AWSClient)(client))
}
