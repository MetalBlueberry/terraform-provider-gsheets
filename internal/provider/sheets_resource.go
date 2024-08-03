package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/api/sheets/v4"
)

var _ resource.ResourceWithConfigure = &SheetResource{}
var _ resource.ResourceWithImportState = &SheetResource{}

func NewSheetResource() resource.Resource {
	return &SheetResource{}
}

type SheetResource struct {
	client *sheets.Service
}

type SpreadsheetPropertiesModel struct {
	Title   types.String `tfsdk:"title"`
	SheetID types.Int64  `tfsdk:"sheet_id"`
	Index   types.Int64  `tfsdk:"index"`
}

type SheetsResourceModel struct {
	SpreadsheetID types.String                `tfsdk:"spreadsheet_id"`
	Properties    *SpreadsheetPropertiesModel `tfsdk:"properties"`
}

func (r *SheetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sheet"
}

func (r *SheetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Sheets resource",

		Attributes: map[string]schema.Attribute{
			"spreadsheet_id": schema.StringAttribute{
				MarkdownDescription: "The file to get the rows from",
				Required:            true,
			},
			"properties": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"title": schema.StringAttribute{
						MarkdownDescription: "The title of the spreadsheet",
						Required:            true,
					},
					"sheet_id": schema.Int64Attribute{
						Computed: true,
					},
					"index": schema.Int64Attribute{
						Computed: true,
					},
				},
			},
		},
	}
}

// Configure implements resource.ResourceWithConfigure.
func (r *SheetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

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
func (r *SheetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SheetsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := r.client.Spreadsheets.BatchUpdate(data.SpreadsheetID.ValueString(), &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: data.Properties.Title.ValueString(),
				},
			}},
		},
	})
	createResponse, err := createRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to create sheet", err.Error())
		return
	}

	data.SpreadsheetID = basetypes.NewStringValue(createResponse.SpreadsheetId)
	data.Properties.Index = basetypes.NewInt64Value(createResponse.Replies[0].AddSheet.Properties.Index)
	data.Properties.SheetID = basetypes.NewInt64Value(createResponse.Replies[0].AddSheet.Properties.SheetId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState implements resource.ResourceWithImportState.
func (r *SheetResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
	panic("unimplemented")
}

// Read is called when the provider must read resource values in order
// to update state. Planned state values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (r *SheetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var data SheetsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// Update is called to update the state of the resource. Config, planned
// state, and prior state values should be read from the
// UpdateRequest and new state values set on the UpdateResponse.
func (r *SheetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData SheetsResourceModel
	var planData SheetsResourceModel

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

	updateRequest := r.client.Spreadsheets.BatchUpdate(stateData.SpreadsheetID.ValueString(), &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						Title: planData.Properties.Title.ValueString(),
					},
				},
			},
		},
	})
	updateResponse, err := updateRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to perform update request", err.Error())
		return
	}

	stateData.SpreadsheetID = basetypes.NewStringValue(updateResponse.SpreadsheetId)
	stateData.Properties.Title = basetypes.NewStringValue(updateResponse.Replies[0].AddSheet.Properties.Title)

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

// Delete is called when the provider must delete the resource. Config
// values may be read from the DeleteRequest.
//
// If execution completes without error, the framework will automatically
// call DeleteResponse.State.RemoveResource(), so it can be omitted
// from provider logic.
func (r *SheetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
