package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/api/sheets/v4"
)

var _ resource.ResourceWithConfigure = &RangeResource{}
var _ resource.ResourceWithImportState = &RangeResource{}

func NewRangeResource() resource.Resource {
	return &RangeResource{}
}

type RangeResource struct {
	client *sheets.Service
}

type RangeResourceModel struct {
	SpreadsheetID    types.String `tfsdk:"spreadsheet_id"`
	Range            types.String `tfsdk:"range"`
	ValueInputOption types.String `tfsdk:"value_input_option"`
	Rows             types.List   `tfsdk:"rows"`
}

func (m RangeResourceModel) RowsToValues() [][]interface{} {
	values := [][]interface{}{}
	for _, el := range m.Rows.Elements() {
		row := []interface{}{}
		for _, ell := range el.(basetypes.ListValue).Elements() {
			row = append(row, ell.(types.String).ValueString())
		}
		values = append(values, row)
	}
	return values
}

func (r *RangeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_range"
}

func (r *RangeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sheets resource",

		Attributes: map[string]schema.Attribute{
			"spreadsheet_id": schema.StringAttribute{
				MarkdownDescription: "The file to get the rows from",
				Required:            true,
			},
			"range": schema.StringAttribute{
				MarkdownDescription: "The range to read",
				Required:            true,
			},
			"value_input_option": schema.StringAttribute{
				MarkdownDescription: "how to post data",
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("USER_ENTERED"),
				Validators: []validator.String{
					stringvalidator.OneOf("RAW", "USER_ENTERED"),
				},
			},
			"rows": schema.ListAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "The rows",
				Optional:            true,
			},
		},
	}
}

// Configure implements resource.ResourceWithConfigure.
func (r *RangeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sheets.Service)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Type",
			fmt.Sprintf("Expected *sheets.Service, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create is called when the provider must create a new resource. Config
// and planned state values should be read from the
// CreateRequest and new state values set on the CreateResponse.
func (r *RangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RangeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := r.client.Spreadsheets.Values.Update(data.SpreadsheetID.ValueString(), data.Range.ValueString(), &sheets.ValueRange{
		Range:  data.Range.ValueString(),
		Values: data.RowsToValues(),
	})
	updateRequest.ValueInputOption(data.ValueInputOption.ValueString())
	_, err := updateRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to perform update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState implements resource.ResourceWithImportState.
func (r *RangeResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
	panic("unimplemented")
}

// Read is called when the provider must read resource values in order
// to update state. Planned state values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (r *RangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var data RangeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// Update is called to update the state of the resource. Config, planned
// state, and prior state values should be read from the
// UpdateRequest and new state values set on the UpdateResponse.
func (r *RangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData RangeResourceModel
	var planData RangeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := r.client.Spreadsheets.Values.Update(stateData.SpreadsheetID.ValueString(), stateData.Range.ValueString(), &sheets.ValueRange{
		Range:  planData.Range.ValueString(),
		Values: planData.RowsToValues(),
	})

	updateRequest.ValueInputOption(planData.ValueInputOption.ValueString())

	updateResponse, err := updateRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to perform update request", err.Error())
		return
	}

	stateData.SpreadsheetID = basetypes.NewStringValue(updateResponse.SpreadsheetId)
	stateData.Rows = planData.Rows
	stateData.ValueInputOption = planData.ValueInputOption

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

// Delete is called when the provider must delete the resource. Config
// values may be read from the DeleteRequest.
//
// If execution completes without error, the framework will automatically
// call DeleteResponse.State.RemoveResource(), so it can be omitted
// from provider logic.
func (r *RangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
