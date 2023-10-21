package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &CruiserProvider{}

// CruiserProvider defines the provider implementation.
type CruiserProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func (p *CruiserProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cruiser"
	resp.Version = p.version
}

func (p *CruiserProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
}

func (p *CruiserProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

}

func (p *CruiserProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRouteResource,
		NewEnvoyClusterResource,
		NewEnvoyClusterLoadAssignmentResource,
		NewEnvoyListenerResource,
		NewEnvoyRouteConfigurationResource,
		NewEnvoyVirtualHostResource,
	}
}

func (p *CruiserProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CruiserProvider{
			version: version,
		}
	}
}
