package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	awspb "github.com/ultraviolet-black/terraform-provider-cruiser/proto/providers/aws"
	serverpb "github.com/ultraviolet-black/terraform-provider-cruiser/proto/server"
	"google.golang.org/protobuf/encoding/protojson"
)

type RouteResource struct {
}

func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

type RouteResourceModel struct {
	Name       types.String `tfsdk:"name"`
	ParentName types.String `tfsdk:"parent_name"`
	Handler    types.Object `tfsdk:"handler"`
	Matchers   types.List   `tfsdk:"matchers"`
	ProtoJson  types.String `tfsdk:"proto_json"`
}

type RouteMatcherResourceModel struct {
	IsGrpcCall    types.Bool   `tfsdk:"is_grpc_call"`
	Host          types.String `tfsdk:"host"`
	Path          types.String `tfsdk:"path"`
	PathPrefix    types.String `tfsdk:"path_prefix"`
	Methods       types.List   `tfsdk:"methods"`
	Schemes       types.List   `tfsdk:"schemes"`
	Headers       types.Map    `tfsdk:"headers"`
	HeadersRegexp types.Map    `tfsdk:"headers_regexp"`
	Queries       types.Map    `tfsdk:"queries"`
}

type RouteHandlerResourceModel struct {
	AwsLambda types.Object `tfsdk:"aws_lambda"`
}

type RouteHandlerAwsLambdaResourceModel struct {
	FunctionName      types.String `tfsdk:"function_name"`
	Qualifier         types.String `tfsdk:"qualifier"`
	EnableHealthCheck types.Bool   `tfsdk:"enable_health_check"`
}

func (r *RouteResourceModel) ToProtobuf(ctx context.Context) (*serverpb.Router_Route, diag.Diagnostics) {

	pb := &serverpb.Router_Route{
		Name:       r.Name.ValueString(),
		ParentName: r.ParentName.ValueString(),
		Handler:    &serverpb.Router_Handler{},
		Matchers:   []*serverpb.Router_Route_Matcher{},
	}

	handlerModel := &RouteHandlerResourceModel{}

	diags := r.Handler.As(ctx, handlerModel, basetypes.ObjectAsOptions{})

	if diags.HasError() {
		return nil, diags
	}

	if !handlerModel.AwsLambda.IsNull() {

		awsLambdaModel := &RouteHandlerAwsLambdaResourceModel{}

		diags.Append(handlerModel.AwsLambda.As(ctx, awsLambdaModel, basetypes.ObjectAsOptions{})...)

		pb.Handler.Backend = &serverpb.Router_Handler_AwsLambda{
			AwsLambda: &awspb.LambdaBackend{
				FunctionName:      awsLambdaModel.FunctionName.ValueString(),
				Qualifier:         awsLambdaModel.Qualifier.ValueString(),
				EnableHealthCheck: awsLambdaModel.EnableHealthCheck.ValueBool(),
			},
		}

	}

	matchers := make([]*RouteMatcherResourceModel, 0, len(r.Matchers.Elements()))

	diags.Append(r.Matchers.ElementsAs(ctx, &matchers, false)...)

	if diags.HasError() {
		return nil, diags
	}

	for _, matcher := range matchers {

		if !matcher.Headers.IsNull() {
			headers := make(map[string]string, len(matcher.Headers.Elements()))
			diags.Append(matcher.Headers.ElementsAs(ctx, &headers, false)...)

			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Headers{
					Headers: &serverpb.Router_Route_HeadersRule{
						Headers: headers,
					},
				},
			})
		} else if !matcher.HeadersRegexp.IsNull() {
			headersRegexp := make(map[string]string, len(matcher.HeadersRegexp.Elements()))
			diags.Append(matcher.HeadersRegexp.ElementsAs(ctx, &headersRegexp, false)...)

			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_HeadersRegexp{
					HeadersRegexp: &serverpb.Router_Route_HeadersRegexpRule{
						HeadersRegexp: headersRegexp,
					},
				},
			})
		} else if !matcher.Queries.IsNull() {
			queries := make(map[string]string, len(matcher.Queries.Elements()))
			diags.Append(matcher.Queries.ElementsAs(ctx, &queries, false)...)

			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Queries{
					Queries: &serverpb.Router_Route_QueriesRule{
						Queries: queries,
					},
				},
			})
		} else if !matcher.Methods.IsNull() {
			methods := make([]string, len(matcher.Methods.Elements()))
			diags.Append(matcher.Methods.ElementsAs(ctx, &methods, false)...)

			methodsEnum := make([]serverpb.Router_Route_MethodsRule_Method, len(methods))

			for i, method := range methods {
				methodsEnum[i] = serverpb.Router_Route_MethodsRule_Method(serverpb.Router_Route_MethodsRule_Method_value[method])
			}

			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Methods{
					Methods: &serverpb.Router_Route_MethodsRule{
						Methods: methodsEnum,
					},
				},
			})
		} else if !matcher.Schemes.IsNull() {
			schemes := make([]string, len(matcher.Schemes.Elements()))
			diags.Append(matcher.Schemes.ElementsAs(ctx, &schemes, false)...)

			schemesEnum := make([]serverpb.Router_Route_SchemesRule_Scheme, len(schemes))

			for i, method := range schemes {
				schemesEnum[i] = serverpb.Router_Route_SchemesRule_Scheme(serverpb.Router_Route_SchemesRule_Scheme_value[method])
			}

			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Schemes{
					Schemes: &serverpb.Router_Route_SchemesRule{
						Schemes: schemesEnum,
					},
				},
			})
		} else if !matcher.Host.IsNull() {
			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Host{
					Host: matcher.Host.ValueString(),
				},
			})
		} else if !matcher.Path.IsNull() {
			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_Path{
					Path: matcher.Path.ValueString(),
				},
			})
		} else if !matcher.PathPrefix.IsNull() {
			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_PathPrefix{
					PathPrefix: matcher.PathPrefix.ValueString(),
				},
			})
		} else if !matcher.IsGrpcCall.IsNull() {
			pb.Matchers = append(pb.Matchers, &serverpb.Router_Route_Matcher{
				Rule: &serverpb.Router_Route_Matcher_IsGrpcCall{
					IsGrpcCall: matcher.IsGrpcCall.ValueBool(),
				},
			})
		}

		if diags.HasError() {
			return nil, diags
		}

	}

	return pb, diags

}

func (r *RouteResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

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

func (r *RouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A Cruiser route",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The JSON representation of the route protobuf",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the resource",
				Required:            true,
			},
			"parent_name": schema.StringAttribute{
				MarkdownDescription: "The name of the parent resource",
				Optional:            true,
			},
			"handler": schema.SingleNestedAttribute{
				MarkdownDescription: "The handler for the resource",
				Required:            true,
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(
						path.MatchRoot("aws_lambda"),
					),
				},
				Attributes: map[string]schema.Attribute{
					"aws_lambda": schema.SingleNestedAttribute{
						MarkdownDescription: "The AWS Lambda handler",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"function_name": schema.StringAttribute{
								MarkdownDescription: "The name of the Lambda function",
								Required:            true,
							},
							"qualifier": schema.StringAttribute{
								MarkdownDescription: "The qualifier of the Lambda function",
								Required:            true,
							},
							"enable_health_check": schema.BoolAttribute{
								MarkdownDescription: "Whether to enable health checks",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
						},
					},
				},
			},
			"matchers": schema.ListNestedAttribute{
				MarkdownDescription: "Matchers for the resource",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{
						objectvalidator.ExactlyOneOf(
							path.MatchRoot("is_grpc_call"),
							path.MatchRoot("host"),
							path.MatchRoot("path"),
							path.MatchRoot("path_prefix"),
							path.MatchRoot("methods"),
							path.MatchRoot("schemes"),
							path.MatchRoot("headers"),
							path.MatchRoot("headers_regexp"),
							path.MatchRoot("queries"),
						),
					},
					Attributes: map[string]schema.Attribute{
						"is_grpc_call": schema.BoolAttribute{
							MarkdownDescription: "Whether the matcher is for a gRPC call",
							Optional:            true,
						},
						"host": schema.StringAttribute{
							MarkdownDescription: "The host to match",
							Optional:            true,
						},
						"path": schema.StringAttribute{
							MarkdownDescription: "The path to match",
							Optional:            true,
						},
						"path_prefix": schema.StringAttribute{
							MarkdownDescription: "The path prefix to match",
							Optional:            true,
						},
						"methods": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "The methods to match",
							Optional:            true,
							Validators: []validator.List{
								listvalidator.ValueStringsAre(
									stringvalidator.OneOf("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "HEAD", "CONNECT", "TRACE"),
								),
							},
						},
						"schemes": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "The schemes to match",
							Optional:            true,
							Validators: []validator.List{
								listvalidator.ValueStringsAre(
									stringvalidator.OneOf("HTTP", "HTTPS"),
								),
							},
						},
						"headers": schema.MapAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "The headers to match",
							Optional:            true,
							Validators: []validator.Map{
								mapvalidator.SizeAtLeast(1),
							},
						},
						"headers_regexp": schema.MapAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "The headers to match using regular expressions",
							Optional:            true,
							Validators: []validator.Map{
								mapvalidator.SizeAtLeast(1),
							},
						},
						"queries": schema.MapAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "The queries to match",
							Optional:            true,
							Validators: []validator.Map{
								mapvalidator.SizeAtLeast(1),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

func (r *RouteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *RouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(RouteResourceModel)

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

func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(RouteResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(RouteResourceModel)

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

func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(RouteResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
