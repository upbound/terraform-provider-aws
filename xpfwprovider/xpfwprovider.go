package xpfwprovider

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
	fwProvider := internalfwprovider.New(p)
	return fwProvider, p, err
}

func GetFrameworkProviderWithPrimary(primary interface {
	Meta() interface{}
}) fwprovider.Provider {
	return internalfwprovider.New(primary)
}

func GetProviderSchema(ctx context.Context) fwschema.Schema {
	p, _ := provider.New(ctx)
	fwProvider := internalfwprovider.New(p)
	schemaReq := fwprovider.SchemaRequest{}
	schemaResp := fwprovider.SchemaResponse{}
	fwProvider.Schema(ctx, schemaReq, &schemaResp)
	return schemaResp.Schema
}

func (ac *AWSConfig) GetClient(ctx context.Context, client *AWSClient) (*conns.AWSClient, diag.Diagnostics) {
	return (*conns.Config)(ac).ConfigureProvider(ctx, (*conns.AWSClient)(client))
}
