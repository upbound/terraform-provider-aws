package xpprovider

import (
	"context"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/provider"
	internalfwprovider "github.com/hashicorp/terraform-provider-aws/internal/provider/fwprovider"
)

type AWSConfig conns.Config
type AWSClient conns.AWSClient

func GetProvider(ctx context.Context) (fwprovider.Provider, *schema.Provider, error) {
	p, err := provider.New(ctx)
	if err != nil {
		return nil, nil, err
	}
	fwProvider := internalfwprovider.New(p)
	return fwProvider, p, err
}

func GetProviderSchema(ctx context.Context) (*schema.Provider, error) {
	return provider.New(ctx)
}

func GetFrameworkProviderSchema(ctx context.Context) (fwschema.Schema, error) {
	fwProvider, _, err := GetProvider(ctx)
	if err != nil {
		return fwschema.Schema{}, err
	}
	schemaReq := fwprovider.SchemaRequest{}
	schemaResp := fwprovider.SchemaResponse{}
	fwProvider.Schema(ctx, schemaReq, &schemaResp)
	return schemaResp.Schema, nil
}

func GetFrameworkProviderWithMeta(primary interface{ Meta() interface{} }) fwprovider.Provider {
	return internalfwprovider.New(primary)
}

func (ac *AWSConfig) GetClient(ctx context.Context, client *AWSClient) (*conns.AWSClient, diag.Diagnostics) {
	return (*conns.Config)(ac).ConfigureProvider(ctx, (*conns.AWSClient)(client))
}
