package provider

import (
	"context"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

type EnvoyRouteConfigurationResource struct {
}

func NewEnvoyRouteConfigurationResource() resource.Resource {
	return &EnvoyRouteConfigurationResource{}
}

type EnvoyRouteConfigurationResourceModel struct {
	Name                     types.String `tfsdk:"name"`
	IgnorePortInHostMatching types.Bool   `tfsdk:"ignore_port_in_host_matching"`
	Vhds                     types.Object `tfsdk:"vhds"`
	ProtoJson                types.String `tfsdk:"proto_json"`
}

type EnvoyVhdsResourceModel struct {
	InitialFetchTimeout       types.String `tfsdk:"initial_fetch_timeout"`
	ResourceApiVersion        types.String `tfsdk:"resource_api_version"`
	ApiType                   types.String `tfsdk:"api_type"`
	TransportApiVersion       types.String `tfsdk:"transport_api_version"`
	SetNodeOnFirstMessageOnly types.Bool   `tfsdk:"set_node_on_first_message_only"`
	EnvoyClusterName          types.String `tfsdk:"envoy_cluster_name"`
}

func (r *EnvoyRouteConfigurationResourceModel) ToProtobuf(ctx context.Context) (*routev3.RouteConfiguration, diag.Diagnostics) {

	pb := &routev3.RouteConfiguration{
		Name:                     r.Name.ValueString(),
		IgnorePortInHostMatching: r.IgnorePortInHostMatching.ValueBool(),
	}

	if !r.Vhds.IsNull() {

		vhdsModel := &EnvoyVhdsResourceModel{}

		diags := r.Vhds.As(ctx, vhdsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, diags
		}

		initialFetchTimeout, err := time.ParseDuration(vhdsModel.InitialFetchTimeout.ValueString())
		if err != nil {
			diags.AddError("InitialFetchTimeout Error", err.Error())
			return nil, diags
		}

		pb.Vhds = &routev3.Vhds{
			ConfigSource: &corev3.ConfigSource{
				InitialFetchTimeout: durationpb.New(initialFetchTimeout),
				ResourceApiVersion:  corev3.ApiVersion(corev3.ApiVersion_value[vhdsModel.ResourceApiVersion.ValueString()]),
				ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
					ApiConfigSource: &corev3.ApiConfigSource{
						ApiType:                   corev3.ApiConfigSource_ApiType(corev3.ApiConfigSource_ApiType_value[vhdsModel.ApiType.ValueString()]),
						TransportApiVersion:       corev3.ApiVersion(corev3.ApiVersion_value[vhdsModel.TransportApiVersion.ValueString()]),
						SetNodeOnFirstMessageOnly: vhdsModel.SetNodeOnFirstMessageOnly.ValueBool(),
						GrpcServices: []*corev3.GrpcService{{
							TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
									ClusterName: vhdsModel.EnvoyClusterName.ValueString(),
								},
							},
						}},
					},
				},
			},
		}

	}

	return pb, []diag.Diagnostic{}

}

func (r *EnvoyRouteConfigurationResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

	routepb, diags := r.ToProtobuf(ctx)

	if diags.HasError() {
		return diags
	}

	jsonpb, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		UseEnumNumbers:  false,
		EmitUnpopulated: false,
	}.Marshal(routepb)
	if err != nil {
		diags.AddError("ProtoJSON Error", err.Error())
		return diags
	}

	r.ProtoJson = basetypes.NewStringValue(string(jsonpb))
	return diags

}

func (r *EnvoyRouteConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Envoy RouteConfiguration Resource",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The Envoy RouteConfiguration Resource in JSON format",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Envoy RouteConfiguration Resource",
				Required:            true,
			},
			"ignore_port_in_host_matching": schema.BoolAttribute{
				MarkdownDescription: "Ignore port in host matching",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"vhds": schema.SingleNestedAttribute{
				MarkdownDescription: "VHDS",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"initial_fetch_timeout": schema.StringAttribute{
						MarkdownDescription: "Initial Fetch Timeout",
						Required:            true,
					},
					"resource_api_version": schema.StringAttribute{
						MarkdownDescription: "The resource api version of the route specifier",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("AUTO"),
						Validators: []validator.String{
							stringvalidator.OneOf("AUTO", "V2", "V3"),
						},
					},
					"api_type": schema.StringAttribute{
						MarkdownDescription: "The api type of the route specifier",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("REST", "GRPC", "DELTA_GRPC", "AGGREGATED_GRPC", "AGGREGATED_DELTA_GRPC"),
						},
					},
					"transport_api_version": schema.StringAttribute{
						MarkdownDescription: "The transport api version of the route specifier",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("AUTO"),
						Validators: []validator.String{
							stringvalidator.OneOf("AUTO", "V2", "V3"),
						},
					},
					"set_node_on_first_message_only": schema.BoolAttribute{
						MarkdownDescription: "The set node on first message only of the route specifier",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"envoy_grpc_cluster_name": schema.StringAttribute{
						MarkdownDescription: "The envoy grpc cluster name of the route specifier",
						Required:            true,
					},
				},
			},
		},
	}
}

func (r *EnvoyRouteConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envoy_route_configuration"
}

func (r *EnvoyRouteConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *EnvoyRouteConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(EnvoyRouteConfigurationResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.LoadProtoJson(ctx)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyRouteConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(EnvoyRouteConfigurationResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyRouteConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(EnvoyRouteConfigurationResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.LoadProtoJson(ctx)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyRouteConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(EnvoyRouteConfigurationResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *EnvoyRouteConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
