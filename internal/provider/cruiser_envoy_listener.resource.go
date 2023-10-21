package provider

import (
	"context"
	"fmt"
	"time"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	streamv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	corsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	grpcwebv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	healthcheckv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	http_connection_managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
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
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type EnvoyListenerResource struct {
}

func NewEnvoyListenerResource() resource.Resource {
	return &EnvoyListenerResource{}
}

type EnvoyListenerResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Address         types.String `tfsdk:"address"`
	AddressProtocol types.String `tfsdk:"address_protocol"`
	AddressPort     types.Number `tfsdk:"address_port"`
	FilterChains    types.List   `tfsdk:"filter_chains"`
	ProtoJson       types.String `tfsdk:"proto_json"`
}

type EnvoyListenerFilterChainResourceModel struct {
	Filters types.List `tfsdk:"filters"`
}

type EnvoyListenerFilterChainFilterResourceModel struct {
	Name                  types.String `tfsdk:"name"`
	HttpConnectionManager types.Object `tfsdk:"http_connection_manager"`
}

type EnvoyHttpConnectionManagerResourceModel struct {
	HttpFilters       types.List   `tfsdk:"http_filters"`
	StatePrefix       types.String `tfsdk:"state_prefix"`
	CodecType         types.String `tfsdk:"codec_type"`
	StreamIdleTimeout types.String `tfsdk:"stream_idle_timeout"`
	AccessLog         types.List   `tfsdk:"access_log"`
	RouteSpecifier    types.Object `tfsdk:"route_specifier"`
}

type EnvoyHttpFilterResourceModel struct {
	HealthCheck types.Object `tfsdk:"health_check"`
	GrpcWeb     types.Object `tfsdk:"grpc_web"`
	Cors        types.Object `tfsdk:"cors"`
	JwtAuthn    types.Object `tfsdk:"jwt_authn"`
}

type EnvoyAccessLogResourceModel struct {
	Output     types.String `tfsdk:"output"`
	TextFormat types.String `tfsdk:"text_format"`
}

type EnvoyRouteSpecifierResourceModel struct {
	RouteConfigName           types.String `tfsdk:"route_config_name"`
	InitialFetchTimeout       types.String `tfsdk:"initial_fetch_timeout"`
	ResourceApiVersion        types.String `tfsdk:"resource_api_version"`
	ApiType                   types.String `tfsdk:"api_type"`
	TransportApiVersion       types.String `tfsdk:"transport_api_version"`
	SetNodeOnFirstMessageOnly types.Bool   `tfsdk:"set_node_on_first_message_only"`
	EnvoyGrpcClusterName      types.String `tfsdk:"envoy_grpc_cluster_name"`
}

type EnvoyHealthCheckResourceModel struct {
	PassThroughMode types.Bool `tfsdk:"pass_through_mode"`
	Headers         types.List `tfsdk:"headers"`
}

type EnvoyHealthCheckHeaderResourceModel struct {
	Name       types.String `tfsdk:"name"`
	ExactMatch types.String `tfsdk:"exact_match"`
}

type EnvoyGrpcWebResourceModel struct {
	Enable types.Bool `tfsdk:"enable"`
}

type EnvoyCorsResourceModel struct {
	Enable types.Bool `tfsdk:"enable"`
}

type EnvoyJwtAuthnResourceModel struct {
	Providers types.List `tfsdk:"providers"`
	Rules     types.List `tfsdk:"rules"`
}

type EnvoyJwtAuthnProviderResourceModel struct {
	Name           types.String `tfsdk:"name"`
	Issuer         types.String `tfsdk:"issuer"`
	Forward        types.Bool   `tfsdk:"forward"`
	Audiences      types.List   `tfsdk:"audiences"`
	FromHeaders    types.List   `tfsdk:"from_headers"`
	ClaimToHeaders types.List   `tfsdk:"claim_to_headers"`
	RemoteJwks     types.Object `tfsdk:"remote_jwks"`
	LocalJwks      types.Object `tfsdk:"local_jwks"`
}

type EnvoyJwtAuthnFromHeaderResourceModel struct {
	Name        types.String `tfsdk:"name"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
}

type EnvoyJwtAuthnClaimToHeaderResourceModel struct {
	HeaderName types.String `tfsdk:"header_name"`
	ClaimName  types.String `tfsdk:"claim_name"`
}

type EnvoyJwtAuthnRemoteJwksResourceModel struct {
	HttpUri             types.String `tfsdk:"http_uri"`
	UpstreamClusterName types.String `tfsdk:"upstream_cluster_name"`
	Timeout             types.String `tfsdk:"timeout"`
}

type EnvoyJwtAuthnLocalJwksResourceModel struct {
	FileName     types.String `tfsdk:"file_name"`
	InlineString types.String `tfsdk:"inline_string"`
}

type EnvoyJwtAuthnRuleResourceModel struct {
	RequiresAnyProvider types.List   `tfsdk:"requires_any_provider"`
	MatchPathPrefix     types.String `tfsdk:"match_path_prefix"`
}

func (r *EnvoyListenerResourceModel) ToProtobuf(ctx context.Context) (*listenerv3.Listener, diag.Diagnostics) {

	portValue, _ := r.AddressPort.ValueBigFloat().Int64()

	pb := &listenerv3.Listener{
		Name: r.Name.ValueString(),
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_Protocol(corev3.SocketAddress_Protocol_value[r.AddressProtocol.ValueString()]),
					Address:  r.Address.ValueString(),
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: uint32(portValue),
					},
				},
			},
		},
		FilterChains: []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{},
			},
		},
	}

	filterChainsModel := make([]*EnvoyListenerFilterChainResourceModel, 0, len(r.FilterChains.Elements()))

	diags := r.FilterChains.ElementsAs(ctx, &filterChainsModel, false)
	if diags.HasError() {
		return nil, diags
	}

	for _, filterChainModel := range filterChainsModel {

		filtersModel := make([]*EnvoyListenerFilterChainFilterResourceModel, 0, len(filterChainModel.Filters.Elements()))

		diags.Append(filterChainModel.Filters.ElementsAs(ctx, &filtersModel, false)...)
		if diags.HasError() {
			return nil, diags
		}

		for _, filterModel := range filtersModel {

			if !filterModel.HttpConnectionManager.IsNull() {

				httpConnModel := &EnvoyHttpConnectionManagerResourceModel{}

				diags.Append(filterModel.HttpConnectionManager.As(ctx, httpConnModel, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return nil, diags
				}

				routeSpecifierModel := &EnvoyRouteSpecifierResourceModel{}

				diags.Append(httpConnModel.RouteSpecifier.As(ctx, routeSpecifierModel, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return nil, diags
				}

				var streamIdleTimeoutPb *durationpb.Duration

				if !httpConnModel.StreamIdleTimeout.IsNull() {

					streamIdleTimeout, err := time.ParseDuration(httpConnModel.StreamIdleTimeout.ValueString())
					if err != nil {
						diags.AddError("StreamIdleTimeout Error", err.Error())
						return nil, diags
					}

					streamIdleTimeoutPb = durationpb.New(streamIdleTimeout)

				}

				initialFetchTimeout, err := time.ParseDuration(routeSpecifierModel.InitialFetchTimeout.ValueString())
				if err != nil {
					diags.AddError("StreamIdleTimeout Error", err.Error())
					return nil, diags
				}

				httpConn := &http_connection_managerv3.HttpConnectionManager{
					CodecType: http_connection_managerv3.HttpConnectionManager_CodecType(http_connection_managerv3.HttpConnectionManager_CodecType_value[httpConnModel.CodecType.ValueString()]),
					RouteSpecifier: &http_connection_managerv3.HttpConnectionManager_Rds{
						Rds: &http_connection_managerv3.Rds{
							RouteConfigName: routeSpecifierModel.RouteConfigName.ValueString(),
							ConfigSource: &corev3.ConfigSource{
								InitialFetchTimeout: durationpb.New(initialFetchTimeout),
								ResourceApiVersion:  corev3.ApiVersion(corev3.ApiVersion_value[routeSpecifierModel.ResourceApiVersion.ValueString()]),
								ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
									ApiConfigSource: &corev3.ApiConfigSource{
										ApiType:                   corev3.ApiConfigSource_ApiType(corev3.ApiConfigSource_ApiType_value[routeSpecifierModel.ApiType.ValueString()]),
										TransportApiVersion:       corev3.ApiVersion(corev3.ApiVersion_value[routeSpecifierModel.TransportApiVersion.ValueString()]),
										SetNodeOnFirstMessageOnly: routeSpecifierModel.SetNodeOnFirstMessageOnly.ValueBool(),
										GrpcServices: []*corev3.GrpcService{
											{
												TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
													EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
														ClusterName: routeSpecifierModel.EnvoyGrpcClusterName.ValueString(),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					StatPrefix:        httpConnModel.StatePrefix.ValueString(),
					AccessLog:         []*accesslogv3.AccessLog{},
					StreamIdleTimeout: streamIdleTimeoutPb,
					HttpFilters:       []*http_connection_managerv3.HttpFilter{},
				}

				if !httpConnModel.AccessLog.IsNull() {

					accessLogsModel := make([]*EnvoyAccessLogResourceModel, 0, len(httpConnModel.AccessLog.Elements()))

					diags.Append(httpConnModel.AccessLog.ElementsAs(ctx, &accessLogsModel, false)...)
					if diags.HasError() {
						return nil, diags
					}

					for _, accessLogModel := range accessLogsModel {

						switch accessLogModel.Output.ValueString() {

						case "STDOUT":
							stdoutAccessLog := &streamv3.StdoutAccessLog{
								AccessLogFormat: &streamv3.StdoutAccessLog_LogFormat{
									LogFormat: &corev3.SubstitutionFormatString{
										Format: &corev3.SubstitutionFormatString_TextFormat{
											TextFormat: accessLogModel.TextFormat.ValueString(),
										},
									},
								},
							}

							accessLogTypedConfig, _ := anypb.New(stdoutAccessLog)

							httpConn.AccessLog = append(httpConn.AccessLog, &accesslogv3.AccessLog{
								Name: "envoy.access_loggers.stdout",
								ConfigType: &accesslogv3.AccessLog_TypedConfig{
									TypedConfig: accessLogTypedConfig,
								},
							})

						}

					}

				}

				routerAny, _ := anypb.New(&routerv3.Router{})

				httpRouter := &http_connection_managerv3.HttpFilter{
					Name: fmt.Sprintf("%s-http-router", filterModel.Name.ValueString()),
					ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
						TypedConfig: routerAny,
					},
				}

				httpConn.HttpFilters = append(httpConn.HttpFilters, httpRouter)

				httpFiltersModel := make([]*EnvoyHttpFilterResourceModel, 0, len(httpConnModel.HttpFilters.Elements()))

				diags.Append(httpConnModel.HttpFilters.ElementsAs(ctx, &httpFiltersModel, false)...)
				if diags.HasError() {
					return nil, diags
				}

				for _, httpFilterModel := range httpFiltersModel {

					if !httpFilterModel.HealthCheck.IsNull() {

						healthCheckModel := &EnvoyHealthCheckResourceModel{}

						diags.Append(httpFilterModel.HealthCheck.As(ctx, healthCheckModel, basetypes.ObjectAsOptions{})...)
						if diags.HasError() {
							return nil, diags
						}

						headersModel := make([]*EnvoyHealthCheckHeaderResourceModel, 0, len(healthCheckModel.Headers.Elements()))

						diags.Append(healthCheckModel.Headers.ElementsAs(ctx, &headersModel, false)...)
						if diags.HasError() {
							return nil, diags
						}

						headers := []*routev3.HeaderMatcher{}

						for _, headerModel := range headersModel {

							headers = append(headers, &routev3.HeaderMatcher{
								Name: headerModel.Name.ValueString(),
								HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
									ExactMatch: headerModel.ExactMatch.ValueString(),
								},
							})

						}

						healthCheckAny, _ := anypb.New(&healthcheckv3.HealthCheck{
							PassThroughMode: wrapperspb.Bool(healthCheckModel.PassThroughMode.ValueBool()),
							Headers:         headers,
						})

						httpConn.HttpFilters = append(httpConn.HttpFilters, &http_connection_managerv3.HttpFilter{
							Name: fmt.Sprintf("%s-health-check", filterModel.Name.ValueString()),
							ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
								TypedConfig: healthCheckAny,
							},
						})

					} else if !httpFilterModel.Cors.IsNull() {

						corsModel := &EnvoyCorsResourceModel{}

						diags.Append(httpFilterModel.Cors.As(ctx, corsModel, basetypes.ObjectAsOptions{})...)
						if diags.HasError() {
							return nil, diags
						}

						if !corsModel.Enable.ValueBool() {
							continue
						}

						corsAny, _ := anypb.New(&corsv3.Cors{})

						httpConn.HttpFilters = append(httpConn.HttpFilters, &http_connection_managerv3.HttpFilter{
							Name: fmt.Sprintf("%s-cors", filterModel.Name.ValueString()),
							ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
								TypedConfig: corsAny,
							},
						})
					} else if !httpFilterModel.GrpcWeb.IsNull() {

						grpcWebModel := &EnvoyGrpcWebResourceModel{}

						diags.Append(httpFilterModel.Cors.As(ctx, grpcWebModel, basetypes.ObjectAsOptions{})...)
						if diags.HasError() {
							return nil, diags
						}

						if !grpcWebModel.Enable.ValueBool() {
							continue
						}

						grpcWebAny, _ := anypb.New(&grpcwebv3.GrpcWeb{})

						httpConn.HttpFilters = append(httpConn.HttpFilters, &http_connection_managerv3.HttpFilter{
							Name: fmt.Sprintf("%s-grpc-web", filterModel.Name.ValueString()),
							ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
								TypedConfig: grpcWebAny,
							},
						})
					} else if !httpFilterModel.JwtAuthn.IsNull() {

						jwtAuthn := &jwtauthnv3.JwtAuthentication{
							Providers: make(map[string]*jwtauthnv3.JwtProvider),
							Rules:     make([]*jwtauthnv3.RequirementRule, 0),
						}

						jwtAuthnModel := &EnvoyJwtAuthnResourceModel{}

						diags.Append(httpFilterModel.JwtAuthn.As(ctx, jwtAuthnModel, basetypes.ObjectAsOptions{})...)
						if diags.HasError() {
							return nil, diags
						}

						providersModel := make([]*EnvoyJwtAuthnProviderResourceModel, 0, len(jwtAuthnModel.Providers.Elements()))

						diags.Append(jwtAuthnModel.Providers.ElementsAs(ctx, &providersModel, false)...)
						if diags.HasError() {
							return nil, diags
						}

						for _, providerModel := range providersModel {

							audiences := make([]string, 0)

							if !providerModel.Audiences.IsNull() {
								audiencesModel := make([]*types.String, 0, len(providerModel.Audiences.Elements()))

								diags.Append(providerModel.Audiences.ElementsAs(ctx, &audiencesModel, false)...)
								if diags.HasError() {
									return nil, diags
								}

								for _, audienceModel := range audiencesModel {

									audiences = append(audiences, audienceModel.ValueString())

								}
							}

							fromHeaders := []*jwtauthnv3.JwtHeader{}

							if !providerModel.FromHeaders.IsNull() {
								fromHeadersModel := make([]*EnvoyJwtAuthnFromHeaderResourceModel, 0, len(providerModel.FromHeaders.Elements()))

								diags.Append(providerModel.FromHeaders.ElementsAs(ctx, &fromHeadersModel, false)...)
								if diags.HasError() {
									return nil, diags
								}

								for _, fromHeaderModel := range fromHeadersModel {

									fromHeaders = append(fromHeaders, &jwtauthnv3.JwtHeader{
										Name:        fromHeaderModel.Name.ValueString(),
										ValuePrefix: fromHeaderModel.ValuePrefix.ValueString(),
									})

								}
							}

							claimToHeaders := []*jwtauthnv3.JwtClaimToHeader{}

							if !providerModel.ClaimToHeaders.IsNull() {
								claimToHeadersModel := make([]*EnvoyJwtAuthnClaimToHeaderResourceModel, 0, len(providerModel.ClaimToHeaders.Elements()))

								diags.Append(providerModel.ClaimToHeaders.ElementsAs(ctx, &claimToHeadersModel, false)...)
								if diags.HasError() {
									return nil, diags
								}

								for _, claimToHeaderModel := range claimToHeadersModel {

									claimToHeaders = append(claimToHeaders, &jwtauthnv3.JwtClaimToHeader{
										HeaderName: claimToHeaderModel.HeaderName.ValueString(),
										ClaimName:  claimToHeaderModel.ClaimName.ValueString(),
									})

								}
							}

							jwtProvider := &jwtauthnv3.JwtProvider{
								Issuer:         providerModel.Issuer.ValueString(),
								Audiences:      audiences,
								Forward:        providerModel.Forward.ValueBool(),
								ClaimToHeaders: claimToHeaders,
								FromHeaders:    fromHeaders,
							}

							if !providerModel.RemoteJwks.IsNull() {

								remoteJwksModel := &EnvoyJwtAuthnRemoteJwksResourceModel{}

								diags.Append(providerModel.RemoteJwks.As(ctx, remoteJwksModel, basetypes.ObjectAsOptions{})...)
								if diags.HasError() {
									return nil, diags
								}

								timeout, err := time.ParseDuration(remoteJwksModel.Timeout.ValueString())
								if err != nil {
									diags.AddError("Timeout Error", err.Error())
									return nil, diags
								}

								jwtProvider.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_RemoteJwks{
									RemoteJwks: &jwtauthnv3.RemoteJwks{
										HttpUri: &corev3.HttpUri{
											Uri:     remoteJwksModel.HttpUri.ValueString(),
											Timeout: durationpb.New(timeout),
											HttpUpstreamType: &corev3.HttpUri_Cluster{
												Cluster: remoteJwksModel.UpstreamClusterName.ValueString(),
											},
										},
									},
								}

							} else if !providerModel.LocalJwks.IsNull() {

								localJwksModel := &EnvoyJwtAuthnLocalJwksResourceModel{}

								diags.Append(providerModel.LocalJwks.As(ctx, localJwksModel, basetypes.ObjectAsOptions{})...)
								if diags.HasError() {
									return nil, diags
								}

								ds := &corev3.DataSource{}

								if !localJwksModel.FileName.IsNull() {

									ds.Specifier = &corev3.DataSource_Filename{
										Filename: localJwksModel.FileName.ValueString(),
									}

								} else if !localJwksModel.InlineString.IsNull() {

									ds.Specifier = &corev3.DataSource_InlineString{
										InlineString: localJwksModel.InlineString.ValueString(),
									}

								}

								jwtProvider.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_LocalJwks{
									LocalJwks: ds,
								}

							}

							jwtAuthn.Providers[providerModel.Name.ValueString()] = jwtProvider

						}

						rulesModel := make([]*EnvoyJwtAuthnRuleResourceModel, 0, len(jwtAuthnModel.Rules.Elements()))

						diags.Append(jwtAuthnModel.Rules.ElementsAs(ctx, &rulesModel, false)...)
						if diags.HasError() {
							return nil, diags
						}

						for _, ruleModel := range rulesModel {

							rule := &jwtauthnv3.RequirementRule{}

							requiresAnyProvider := make([]string, 0, len(ruleModel.RequiresAnyProvider.Elements()))

							diags.Append(ruleModel.RequiresAnyProvider.ElementsAs(ctx, &requiresAnyProvider, false)...)
							if diags.HasError() {
								return nil, diags
							}

							reqs := []*jwtauthnv3.JwtRequirement{}

							for _, providerName := range requiresAnyProvider {
								reqs = append(reqs, &jwtauthnv3.JwtRequirement{
									RequiresType: &jwtauthnv3.JwtRequirement_ProviderName{
										ProviderName: providerName,
									},
								})
							}

							rule.RequirementType = &jwtauthnv3.RequirementRule_Requires{
								Requires: &jwtauthnv3.JwtRequirement{
									RequiresType: &jwtauthnv3.JwtRequirement_RequiresAny{
										RequiresAny: &jwtauthnv3.JwtRequirementOrList{
											Requirements: reqs,
										},
									},
								},
							}

							rule.Match = &routev3.RouteMatch{
								PathSpecifier: &routev3.RouteMatch_Prefix{
									Prefix: ruleModel.MatchPathPrefix.ValueString(),
								},
								CaseSensitive: wrapperspb.Bool(false),
							}

							jwtAuthn.Rules = append(jwtAuthn.Rules, rule)

						}

						jwtAuthnAny, _ := anypb.New(jwtAuthn)

						httpConn.HttpFilters = append(httpConn.HttpFilters, &http_connection_managerv3.HttpFilter{
							Name: fmt.Sprintf("%s-jwt-authn", filterModel.Name.ValueString()),
							ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
								TypedConfig: jwtAuthnAny,
							},
						})

					}

				}

				httpConnAny, _ := anypb.New(httpConn)

				pb.FilterChains[0].Filters = append(pb.FilterChains[0].Filters, &listenerv3.Filter{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listenerv3.Filter_TypedConfig{
						TypedConfig: httpConnAny,
					},
				})
			}

		}

	}

	return pb, []diag.Diagnostic{}

}

func (r *EnvoyListenerResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

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

func (r *EnvoyListenerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Envoy Listener Resource",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The Envoy Listener Resource in JSON format",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Envoy Listener Resource",
				Required:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The address of the Envoy Listener Resource",
				Required:            true,
			},
			"address_protocol": schema.StringAttribute{
				MarkdownDescription: "The address protocol of the Envoy Listener Resource",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("TCP", "UDP"),
				},
			},
			"address_port": schema.NumberAttribute{
				MarkdownDescription: "The address port of the Envoy Listener Resource",
				Required:            true,
			},
			"filter_chains": schema.ListNestedAttribute{
				MarkdownDescription: "The filter chains of the Envoy Listener Resource",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"filters": schema.ListNestedAttribute{
							MarkdownDescription: "The filters of the Envoy Listener Resource",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the filter",
										Required:            true,
									},
									"http_connection_manager": schema.SingleNestedAttribute{
										MarkdownDescription: "The http connection manager of the filter",
										Required:            true,
										Attributes: map[string]schema.Attribute{
											"state_prefix": schema.StringAttribute{
												MarkdownDescription: "The state prefix of the http connection manager",
												Optional:            true,
											},
											"codec_type": schema.StringAttribute{
												MarkdownDescription: "The codec type of the http connection manager",
												Optional:            true,
												Computed:            true,
												Default:             stringdefault.StaticString("AUTO"),
												Validators: []validator.String{
													stringvalidator.OneOf("AUTO", "HTTP1", "HTTP2", "HTTP3"),
												},
											},
											"stream_idle_timeout": schema.StringAttribute{
												MarkdownDescription: "The stream idle timeout of the http connection manager",
												Required:            true,
											},
											"access_log": schema.ListNestedAttribute{
												MarkdownDescription: "The access log of the http connection manager",
												Optional:            true,
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"output": schema.StringAttribute{
															MarkdownDescription: "The output of the access log",
															Optional:            true,
															Computed:            true,
															Default:             stringdefault.StaticString("STDOUT"),
															Validators: []validator.String{
																stringvalidator.OneOf("STDOUT"),
															},
														},
														"text_format": schema.StringAttribute{
															MarkdownDescription: "The text format of the access log",
															Required:            true,
														},
													},
												},
											},
											"route_specifier": schema.SingleNestedAttribute{
												MarkdownDescription: "The route specifier of the http connection manager",
												Required:            true,
												Attributes: map[string]schema.Attribute{
													"route_config_name": schema.StringAttribute{
														MarkdownDescription: "The route config name of the route specifier",
														Required:            true,
													},
													"initial_fetch_timeout": schema.StringAttribute{
														MarkdownDescription: "The initial fetch timeout of the route specifier",
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
											"http_filters": schema.ListNestedAttribute{
												MarkdownDescription: "The http filters of the http connection manager",
												Required:            true,
												NestedObject: schema.NestedAttributeObject{
													Validators: []validator.Object{
														objectvalidator.ExactlyOneOf(
															path.MatchRoot("health_check"),
															path.MatchRoot("cors"),
															path.MatchRoot("grpc_web"),
															path.MatchRoot("jwt_authn"),
														),
													},
													Attributes: map[string]schema.Attribute{
														"health_check": schema.SingleNestedAttribute{
															MarkdownDescription: "The health check of the http filter",
															Optional:            true,
															Attributes: map[string]schema.Attribute{
																"pass_through_mode": schema.BoolAttribute{
																	MarkdownDescription: "The pass through mode of the health check",
																	Optional:            true,
																	Computed:            true,
																	Default:             booldefault.StaticBool(false),
																},
																"headers": schema.ListNestedAttribute{
																	MarkdownDescription: "The headers of the health check",
																	Required:            true,
																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: "The name of the header",
																				Required:            true,
																			},
																			"exact_match": schema.StringAttribute{
																				MarkdownDescription: "The exact match of the header",
																				Required:            true,
																			},
																		},
																	},
																},
															},
														},
														"cors": schema.SingleNestedAttribute{
															MarkdownDescription: "The cors of the http filter",
															Optional:            true,
															Attributes: map[string]schema.Attribute{
																"enable": schema.BoolAttribute{
																	MarkdownDescription: "The enable of the cors",
																	Optional:            true,
																	Computed:            true,
																	Default:             booldefault.StaticBool(true),
																},
															},
														},
														"grpc_web": schema.SingleNestedAttribute{
															MarkdownDescription: "The grpc web of the http filter",
															Optional:            true,
															Attributes: map[string]schema.Attribute{
																"enable": schema.BoolAttribute{
																	MarkdownDescription: "The enable of the grpc web",
																	Optional:            true,
																	Computed:            true,
																	Default:             booldefault.StaticBool(true),
																},
															},
														},
														"jwt_authn": schema.SingleNestedAttribute{
															MarkdownDescription: "The jwt authn of the http filter",
															Optional:            true,
															Attributes: map[string]schema.Attribute{
																"providers": schema.ListNestedAttribute{
																	MarkdownDescription: "The providers of the jwt authn",
																	Required:            true,
																	NestedObject: schema.NestedAttributeObject{
																		Validators: []validator.Object{
																			objectvalidator.ExactlyOneOf(
																				path.MatchRoot("remote_jwks"),
																				path.MatchRoot("local_jwks"),
																			),
																		},
																		Attributes: map[string]schema.Attribute{
																			"name": schema.StringAttribute{
																				MarkdownDescription: "The name of the provider",
																				Required:            true,
																			},
																			"issuer": schema.StringAttribute{
																				MarkdownDescription: "The issuer of the provider",
																				Optional:            true,
																			},
																			"audiences": schema.ListAttribute{
																				MarkdownDescription: "The audiences of the provider",
																				Optional:            true,
																				ElementType:         types.StringType,
																			},
																			"forward": schema.BoolAttribute{
																				MarkdownDescription: "The forward of the provider",
																				Optional:            true,
																				Computed:            true,
																				Default:             booldefault.StaticBool(false),
																			},
																			"from_headers": schema.ListNestedAttribute{
																				MarkdownDescription: "The from headers of the provider",
																				Optional:            true,
																				NestedObject: schema.NestedAttributeObject{
																					Attributes: map[string]schema.Attribute{
																						"name": schema.StringAttribute{
																							MarkdownDescription: "The name of the from header",
																							Required:            true,
																						},
																						"value_prefix": schema.StringAttribute{
																							MarkdownDescription: "The value prefix of the from header",
																							Required:            true,
																						},
																					},
																				},
																			},
																			"claim_to_headers": schema.ListNestedAttribute{
																				MarkdownDescription: "The claim to headers of the provider",
																				Optional:            true,
																				NestedObject: schema.NestedAttributeObject{
																					Attributes: map[string]schema.Attribute{
																						"header_name": schema.StringAttribute{
																							MarkdownDescription: "The header name of the claim to header",
																							Required:            true,
																						},
																						"claim_name": schema.StringAttribute{
																							MarkdownDescription: "The claim name of the claim to header",
																							Required:            true,
																						},
																					},
																				},
																			},
																			"remote_jwks": schema.SingleNestedAttribute{
																				MarkdownDescription: "The remote jwks of the provider",
																				Optional:            true,
																				Attributes: map[string]schema.Attribute{
																					"http_uri": schema.StringAttribute{
																						MarkdownDescription: "The http uri of the remote jwks",
																						Required:            true,
																					},
																					"timeout": schema.StringAttribute{
																						MarkdownDescription: "The timeout of the remote jwks",
																						Required:            true,
																					},
																					"upstream_cluster_name": schema.StringAttribute{
																						MarkdownDescription: "The upstream cluster name of the remote jwks",
																						Required:            true,
																					},
																				},
																			},
																			"local_jwks": schema.SingleNestedAttribute{
																				MarkdownDescription: "The local jwks of the provider",
																				Optional:            true,
																				Validators: []validator.Object{
																					objectvalidator.ExactlyOneOf(
																						path.MatchRoot("file_name"),
																						path.MatchRoot("inline_string"),
																					),
																				},
																				Attributes: map[string]schema.Attribute{
																					"file_name": schema.StringAttribute{
																						MarkdownDescription: "The file name of the local jwks",
																						Optional:            true,
																					},
																					"inline_string": schema.StringAttribute{
																						MarkdownDescription: "The inline string of the local jwks",
																						Optional:            true,
																					},
																				},
																			},
																		},
																	},
																},
																"rules": schema.ListNestedAttribute{
																	MarkdownDescription: "The rules of the jwt authn",
																	Required:            true,
																	NestedObject: schema.NestedAttributeObject{
																		Attributes: map[string]schema.Attribute{
																			"match_path_prefix": schema.StringAttribute{
																				MarkdownDescription: "The match path prefix of the rule",
																				Required:            true,
																			},
																			"requires_any_provider": schema.ListAttribute{
																				MarkdownDescription: "The requires any provider of the rule",
																				Required:            true,
																				ElementType:         types.StringType,
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *EnvoyListenerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envoy_listener"
}

func (r *EnvoyListenerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *EnvoyListenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(EnvoyListenerResourceModel)

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

func (r *EnvoyListenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(EnvoyListenerResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyListenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(EnvoyListenerResourceModel)

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

func (r *EnvoyListenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(EnvoyListenerResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *EnvoyListenerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
