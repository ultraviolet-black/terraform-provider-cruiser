package provider

import (
	"context"

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/encoding/protojson"
)

type EnvoyClusterLoadAssignmentResource struct {
}

func NewEnvoyClusterLoadAssignmentResource() resource.Resource {
	return &EnvoyClusterLoadAssignmentResource{}
}

type EnvoyClusterLoadAssignmentResourceModel struct {
	Name        types.String `tfsdk:"name"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ProtoJson   types.String `tfsdk:"proto_json"`
}

func (r *EnvoyClusterLoadAssignmentResourceModel) ToProtobuf(ctx context.Context) (*endpointv3.ClusterLoadAssignment, diag.Diagnostics) {

	pb := &endpointv3.ClusterLoadAssignment{
		ClusterName: r.ClusterName.ValueString(),
	}

	return pb, []diag.Diagnostic{}

}

func (r *EnvoyClusterLoadAssignmentResourceModel) LoadProtoJson(ctx context.Context) diag.Diagnostics {

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

func (r *EnvoyClusterLoadAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Envoy ClusterLoadAssignment Resource",
		Attributes: map[string]schema.Attribute{
			"proto_json": schema.StringAttribute{
				MarkdownDescription: "The Envoy ClusterLoadAssignment Resource in JSON format",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Envoy ClusterLoadAssignment Resource",
				Required:            true,
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "The name of the Envoy Cluster Resource",
				Required:            true,
			},
		},
	}
}

func (r *EnvoyClusterLoadAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_envoy_cluster_load_assignment"
}

func (r *EnvoyClusterLoadAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

}

func (r *EnvoyClusterLoadAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := new(EnvoyClusterLoadAssignmentResourceModel)

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

func (r *EnvoyClusterLoadAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := new(EnvoyClusterLoadAssignmentResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *EnvoyClusterLoadAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := new(EnvoyClusterLoadAssignmentResourceModel)

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

func (r *EnvoyClusterLoadAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := new(EnvoyClusterLoadAssignmentResourceModel)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, data)...)

}

func (r *EnvoyClusterLoadAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
