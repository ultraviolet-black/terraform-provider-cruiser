package provider

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"time"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

type EnvoyVirtualHostResource struct {
}

func NewEnvoyVirtualHostResource() resource.Resource {
	return &EnvoyVirtualHostResource{}
}

type EnvoyVirtualHostResourceModel struct {
	Name      types.String `tfsdk:"name"`
	Domains   types.List   `tfsdk:"domains"`
	Routes    types.List   `tfsdk:"routes"`
	Cors      types.Object `tfsdk:"cors"`
	ProtoJson types.String `tfsdk:"proto_json"`
}

type EnvoyVirtualHostCorsResourceModel struct {
	AllowOriginStringMatchPrefix types.List   `tfsdk:"allow_origin_string_match_prefix"`
	AllowHeaders                 types.List   `tfsdk:"allow_headers"`
	AllowMethods                 types.List   `tfsdk:"allow_methods"`
	ExposeHeaders                types.List   `tfsdk:"expose_headers"`
	MaxAge                       types.Number `tfsdk:"max_age"`
}

type EnvoyVirtualHostRouteResourceModel struct {
	MatchPathPrefix types.String `tfsdk:"match_path_prefix"`
	RouteAction     types.Object `tfsdk:"route_action"`
}

type EnvoyRouteActionResourceModel struct {
	Timeout              types.String `tfsdk:"timeout"`
	ClusterName          types.String `tfsdk:"cluster_name"`
	PrefixRewrite        types.String `tfsdk:"prefix_rewrite"`
	GrpcTimeoutHeaderMax types.String `tfsdk:"grpc_timeout_header_max"`
}

func (r *EnvoyVirtualHostResourceModel) ToProtobuf(ctx context.Context) (*routev3.VirtualHost, diag.Diagnostics) {

	pb := &routev3.VirtualHost{
		Name:   r.Name.ValueString(),
		Routes: []*routev3.Route{},
	}

	if !r.Domains.IsNull() {

		domains := make([]string, 0, len(r.Domains.Elements()))

		diags := r.Domains.ElementsAs(ctx, &domains, false)
		if diags.HasError() {
			return nil, diags
		}

		pb.Domains = domains

	}

	if !r.Cors.IsNull() {

		corsModel := &EnvoyVirtualHostCorsResourceModel{}

		diags := r.Cors.As(ctx, corsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, diags
		}

		pb.Cors = &routev3.CorsPolicy{}

		if !corsModel.AllowHeaders.IsNull() {

			allowHeaders := make([]string, 0, len(corsModel.AllowHeaders.Elements()))

			diags := corsModel.AllowHeaders.ElementsAs(ctx, &allowHeaders, false)
			if diags.HasError() {
				return nil, diags
			}

			pb.Cors.AllowHeaders = strings.Join(allowHeaders, ",")

		}

		if !corsModel.AllowMethods.IsNull() {

			allowMethods := make([]string, 0, len(corsModel.AllowMethods.Elements()))

			diags := corsModel.AllowMethods.ElementsAs(ctx, &allowMethods, false)
			if diags.HasError() {
				return nil, diags
			}

			pb.Cors.AllowMethods = strings.Join(allowMethods, ",")

		}

		if !corsModel.AllowOriginStringMatchPrefix.IsNull() {

			allowOriginStringMatchPrefix := make([]string, 0, len(corsModel.AllowOriginStringMatchPrefix.Elements()))

			diags := corsModel.AllowOriginStringMatchPrefix.ElementsAs(ctx, &allowOriginStringMatchPrefix, false)
			if diags.HasError() {
				return nil, diags
			}

			pb.Cors.AllowOriginStringMatch = []*matcherv3.StringMatcher{}

			for _, allowOriginStringMatchPrefix := range allowOriginStringMatchPrefix {

				pb.Cors.AllowOriginStringMatch = append(pb.Cors.AllowOriginStringMatch, &matcherv3.StringMatcher{
					MatchPattern: &matcherv3.StringMatcher_Prefix{
						Prefix: allowOriginStringMatchPrefix,
					},
				})

			}

		}

		if !corsModel.ExposeHeaders.IsNull() {

			exposeHeaders := make([]string, 0, len(corsModel.ExposeHeaders.Elements()))

			diags := corsModel.ExposeHeaders.ElementsAs(ctx, &exposeHeaders, false)
			if diags.HasError() {
				return nil, diags
			}

			pb.Cors.ExposeHeaders = strings.Join(exposeHeaders, ",")

		}

		if !corsModel.MaxAge.IsNull() {

			maxAge, _ := corsModel.MaxAge.ValueBigFloat().Int64()

			pb.Cors.MaxAge = strconv.FormatInt(maxAge, 10)

		}

	}

	if !r.Routes.IsNull() {

		routesModel := make([]*EnvoyVirtualHostRouteResourceModel, 0, len(r.Routes.Elements()))

		diags := r.Routes.ElementsAs(ctx, &routesModel, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, routeModel := range routesModel {

			routeActionModel := &EnvoyRouteActionResourceModel{}

			diags := routeModel.RouteAction.As(ctx, routeActionModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			timeout, err := time.ParseDuration(routeActionModel.Timeout.ValueString())
			if err != nil {
				diags.AddError("Timeout Error", err.Error())
				return nil, diags
			}

			grpcTimeoutHeaderMax, err := time.ParseDuration(routeActionModel.GrpcTimeoutHeaderMax.ValueString())
			if err != nil {
				diags.AddError("Timeout Error", err.Error())
				return nil, diags
			}

			pb.Routes = append(pb.Routes, &routev3.Route{
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Prefix{
						Prefix: routeModel.MatchPathPrefix.ValueString(),
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						Timeout: durationpb.New(timeout),
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: routeActionModel.ClusterName.ValueString(),
						},
						PrefixRewrite: routeActionModel.PrefixRewrite.ValueString(),
						MaxStreamDuration: &routev3.RouteAction_MaxStreamDuration{
							GrpcTimeoutHeaderMax: durationpb.New(grpcTimeoutHeaderMax),
						},
					},
				},
			})

		}

	}

	return pb, []diag.Diagnostic{}

}

func (r *EnvoyVirtualHostResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

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

func (r *EnvoyVirtualHostResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Envoy VirtualHost Resource",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The Envoy VirtualHost Resource in JSON format",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Envoy VirtualHost Resource",
				Required:            true,
			},
			"domains": schema.ListAttribute{
				MarkdownDescription: "The domains of the Envoy VirtualHost Resource",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"routes": schema.ListNestedAttribute{
				MarkdownDescription: "The routes of the Envoy VirtualHost Resource",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"match_path_prefix": schema.StringAttribute{
							MarkdownDescription: "The match path prefix of the Envoy VirtualHost Resource",
							Required:            true,
						},
						"route_action": schema.SingleNestedAttribute{
							MarkdownDescription: "The route action of the Envoy VirtualHost Resource",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"timeout": schema.StringAttribute{
									MarkdownDescription: "The timeout of the Envoy VirtualHost Resource",
									Required:            true,
								},
								"cluster_name": schema.StringAttribute{
									MarkdownDescription: "The cluster name of the Envoy VirtualHost Resource",
									Required:            true,
								},
								"prefix_rewrite": schema.StringAttribute{
									MarkdownDescription: "The prefix rewrite of the Envoy VirtualHost Resource",
									Optional:            true,
								},
								"grpc_timeout_header_max": schema.StringAttribute{
									MarkdownDescription: "The grpc timeout header max of the Envoy VirtualHost Resource",
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("0s"),
								},
							},
						},
					},
				},
			},
			"cors": schema.SingleNestedAttribute{
				MarkdownDescription: "The cors of the Envoy VirtualHost Resource",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"allow_origin_string_match_prefix": schema.ListAttribute{
						MarkdownDescription: "The allow origin string match prefix of the Envoy VirtualHost Resource",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"allow_headers": schema.ListAttribute{
						MarkdownDescription: "The allow headers of the Envoy VirtualHost Resource",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"allow_methods": schema.ListAttribute{
						MarkdownDescription: "The allow methods of the Envoy VirtualHost Resource",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"expose_headers": schema.ListAttribute{
						MarkdownDescription: "The expose headers of the Envoy VirtualHost Resource",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"max_age": schema.NumberAttribute{
						MarkdownDescription: "The max age of the Envoy VirtualHost Resource",
						Optional:            true,
						Computed:            true,
						Default:             numberdefault.StaticBigFloat(big.NewFloat(0)),
					},
				},
			},
		},
	}
}

func (r *EnvoyVirtualHostResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envoy_virtual_host"
}

func (r *EnvoyVirtualHostResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *EnvoyVirtualHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(EnvoyVirtualHostResourceModel)

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

func (r *EnvoyVirtualHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(EnvoyVirtualHostResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyVirtualHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(EnvoyVirtualHostResourceModel)

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

func (r *EnvoyVirtualHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(EnvoyVirtualHostResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *EnvoyVirtualHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
