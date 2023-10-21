package provider

import (
	"context"
	"math/big"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type EnvoyClusterResource struct {
}

func NewEnvoyClusterResource() resource.Resource {
	return &EnvoyClusterResource{}
}

type EnvoyClusterResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	ClusterDiscoveryType types.String `tfsdk:"cluster_discovery_type"`
	LbPolicy             types.String `tfsdk:"lb_policy"`
	ConnectTimeout       types.String `tfsdk:"connect_timeout"`
	DnsLookupFamily      types.String `tfsdk:"dns_lookup_family"`
	EdsClusterConfig     types.Object `tfsdk:"eds_cluster_config"`
	ProtocolOptions      types.Object `tfsdk:"protocol_options"`
	EnableTlsTransport   types.Bool   `tfsdk:"enable_tls_transport"`
	ProtoJson            types.String `tfsdk:"proto_json"`
}

type EnvoyClusterEdsClusterConfigResourceModel struct {
	ServiceName               types.String `tfsdk:"service_name"`
	ResourceApi               types.Object `tfsdk:"resource_api"`
	ApiType                   types.String `tfsdk:"api_type"`
	TransportApiVersion       types.String `tfsdk:"transport_api_version"`
	SetNodeOnFirstMessageOnly types.Bool   `tfsdk:"set_node_on_first_message_only"`
	EnvoyGrpcClusterName      types.String `tfsdk:"envoy_grpc_cluster_name"`
}

type EnvoyClusterProtocolOptionsResourceModel struct {
	EnableHttp1               types.Bool   `tfsdk:"enable_http1"`
	EnableHttp2               types.Bool   `tfsdk:"enable_http2"`
	Http2MaxConcurrentStreams types.Number `tfsdk:"http2_max_concurrent_streams"`
}

func (r *EnvoyClusterResourceModel) ToProtobuf(ctx context.Context) (*clusterv3.Cluster, diag.Diagnostics) {

	connectTimeout, err := time.ParseDuration(r.ConnectTimeout.String())
	if err != nil {
		return nil, []diag.Diagnostic{diag.NewErrorDiagnostic("Unable to parse connect timeout", err.Error())}
	}

	pb := &clusterv3.Cluster{
		Name: r.Name.String(),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{
			Type: clusterv3.Cluster_DiscoveryType(clusterv3.Cluster_DiscoveryType_value[r.ClusterDiscoveryType.String()]),
		},
		LbPolicy:                      clusterv3.Cluster_LbPolicy(clusterv3.Cluster_LbPolicy_value[r.LbPolicy.String()]),
		ConnectTimeout:                durationpb.New(connectTimeout),
		DnsLookupFamily:               clusterv3.Cluster_DnsLookupFamily(clusterv3.Cluster_DnsLookupFamily_value[r.DnsLookupFamily.String()]),
		TypedExtensionProtocolOptions: map[string]*anypb.Any{},
	}

	if r.EnableTlsTransport.ValueBool() {

		tlsTransport, err := anypb.New(&tlsv3.UpstreamTlsContext{})
		if err != nil {
			return nil, []diag.Diagnostic{diag.NewErrorDiagnostic("Unable to convert UpstreamTlsContext to Any", err.Error())}
		}

		pb.TransportSocket = &corev3.TransportSocket{
			Name: "envoy.transport_sockets.tls",
			ConfigType: &corev3.TransportSocket_TypedConfig{
				TypedConfig: tlsTransport,
			},
		}
	}

	if !r.EdsClusterConfig.IsNull() {

		clusterConfig := &EnvoyClusterEdsClusterConfigResourceModel{}

		diags := r.EdsClusterConfig.As(ctx, clusterConfig, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, diags
		}

		pb.EdsClusterConfig = &clusterv3.Cluster_EdsClusterConfig{
			ServiceName: clusterConfig.ServiceName.String(),
			EdsConfig: &corev3.ConfigSource{
				ResourceApiVersion: corev3.ApiVersion(corev3.ApiVersion_value[clusterConfig.ResourceApi.String()]),
				ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
					ApiConfigSource: &corev3.ApiConfigSource{
						ApiType:                   corev3.ApiConfigSource_ApiType(corev3.ApiConfigSource_ApiType_value[clusterConfig.ApiType.String()]),
						TransportApiVersion:       corev3.ApiVersion(corev3.ApiVersion_value[clusterConfig.TransportApiVersion.String()]),
						SetNodeOnFirstMessageOnly: clusterConfig.SetNodeOnFirstMessageOnly.ValueBool(),
						GrpcServices: []*corev3.GrpcService{{
							TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
									ClusterName: clusterConfig.EnvoyGrpcClusterName.String(),
								},
							},
						}},
					},
				},
			},
		}

	}

	if !r.ProtocolOptions.IsNull() {

		protocolOptions := &EnvoyClusterProtocolOptionsResourceModel{}

		diags := r.ProtocolOptions.As(ctx, &protocolOptions, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, diags
		}

		if protocolOptions.EnableHttp1.ValueBool() {
			protocolOptsAny, err := anypb.New(&httpv3.HttpProtocolOptions{
				UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
					ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
						ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
							HttpProtocolOptions: &corev3.Http1ProtocolOptions{},
						},
					},
				},
			})
			if err != nil {
				return nil, []diag.Diagnostic{diag.NewErrorDiagnostic("Unable to convert HttpProtocolOptions to Any", err.Error())}
			}

			pb.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"] = protocolOptsAny
		}

		if protocolOptions.EnableHttp2.ValueBool() {

			maxConcurrentStreams, _ := protocolOptions.Http2MaxConcurrentStreams.ValueBigFloat().Int64()

			protocolOptsAny, err := anypb.New(&httpv3.HttpProtocolOptions{
				UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
					ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
						ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{
							Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
								MaxConcurrentStreams: wrapperspb.UInt32(uint32(maxConcurrentStreams)),
							},
						},
					},
				},
			})
			if err != nil {
				return nil, []diag.Diagnostic{diag.NewErrorDiagnostic("Unable to convert HttpProtocolOptions to Any", err.Error())}
			}

			pb.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"] = protocolOptsAny
		}

	}

	return pb, []diag.Diagnostic{}

}

func (r *EnvoyClusterResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

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

func (r *EnvoyClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Envoy Cluster Resource",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The Envoy Cluster Resource in JSON format",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster. This name must match the name used in the Envoy proxy configuration.",
				Required:            true,
			},
			"cluster_discovery_type": schema.StringAttribute{
				MarkdownDescription: "The type of discovery service that will be used to discover cluster members. Valid values are `EDS`, `STATIC`, `STRICT_DNS`, `LOGICAL_DNS` and `ORIGINAL_DST`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("EDS"),
				Validators: []validator.String{
					stringvalidator.OneOf("EDS", "STATIC", "STRICT_DNS", "LOGICAL_DNS", "ORIGINAL_DST"),
				},
			},
			"lb_policy": schema.StringAttribute{
				MarkdownDescription: "The load balancing policy for the cluster. Valid values are `ROUND_ROBIN`, `LEAST_REQUEST`, `RANDOM`, `RING_HASH`, `MAGLEV`, `CLUSTER_PROVIDED`, `LOAD_BALANCING_POLICY_CONFIG`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("ROUND_ROBIN"),
				Validators: []validator.String{
					stringvalidator.OneOf("ROUND_ROBIN", "LEAST_REQUEST", "RANDOM", "RING_HASH", "MAGLEV", "CLUSTER_PROVIDED", "LOAD_BALANCING_POLICY_CONFIG"),
				},
			},
			"connect_timeout": schema.StringAttribute{
				MarkdownDescription: "The timeout for new network connections to hosts in the cluster. If not specified, the default is 5s.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("5s"),
			},
			"dns_lookup_family": schema.StringAttribute{
				MarkdownDescription: "The DNS resolver to use for hostname resolution. Valid values are `V4_ONLY`, `V6_ONLY`, `V4_PREFERRED`, `ALL` and `AUTO`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("AUTO"),
				Validators: []validator.String{
					stringvalidator.OneOf("V4_ONLY", "V6_ONLY", "V4_PREFERRED", "ALL", "AUTO"),
				},
			},
			"enable_tls_transport": schema.BoolAttribute{
				MarkdownDescription: "Whether to enable TLS transport for the cluster.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"eds_cluster_config": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration for the EDS cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"service_name": schema.StringAttribute{
						MarkdownDescription: "The name of the service to load balance. This name must match the name used in the Envoy proxy configuration.",
						Required:            true,
					},
					"resource_api": schema.StringAttribute{
						MarkdownDescription: "The resource API version to use. Valid values are `AUTO`, `V2` and `V3`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("V3"),
						Validators: []validator.String{
							stringvalidator.OneOf("AUTO", "V2", "V3"),
						},
					},
					"api_type": schema.StringAttribute{
						MarkdownDescription: "The API type to use. Valid values are `GRPC`, `REST`, `DELTA_GRPC`, `AGGREGATED_GRPC` and `AGGREGATED_DELTA_GRPC`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("GRPC"),
						Validators: []validator.String{
							stringvalidator.OneOf("GRPC", "REST", "DELTA_GRPC", "AGGREGATED_GRPC", "AGGREGATED_DELTA_GRPC"),
						},
					},
					"transport_api_version": schema.StringAttribute{
						MarkdownDescription: "The transport API version to use. Valid values are `AUTO`, `V2` and `V3`.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("AUTO"),
						Validators: []validator.String{
							stringvalidator.OneOf("AUTO", "V2", "V3"),
						},
					},
					"set_node_on_first_message_only": schema.BoolAttribute{
						MarkdownDescription: "Whether to set the node on the first message only.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"envoy_grpc_cluster_name": schema.StringAttribute{
						MarkdownDescription: "The name of the Envoy gRPC cluster.",
						Optional:            true,
					},
				},
			},
			"protocol_options": schema.SingleNestedAttribute{
				MarkdownDescription: "The protocol options for the cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"enable_http1": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable HTTP/1.1.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"enable_http2": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable HTTP/2.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"http2_max_concurrent_streams": schema.NumberAttribute{
						MarkdownDescription: "The maximum number of concurrent HTTP/2 streams.",
						Optional:            true,
						Computed:            true,
						Default:             numberdefault.StaticBigFloat(big.NewFloat(2147483647)),
					},
				},
			},
		},
	}
}

func (r *EnvoyClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envoy_cluster"
}

func (r *EnvoyClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *EnvoyClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(EnvoyClusterResourceModel)

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

func (r *EnvoyClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(EnvoyClusterResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(EnvoyClusterResourceModel)

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

func (r *EnvoyClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(EnvoyClusterResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *EnvoyClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
