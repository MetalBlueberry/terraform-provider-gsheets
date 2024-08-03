package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	for i := range values {
		values[i] = removeTrailingEmptyStrings(values[i])
	}
	values = removeEmptyRows(values)

	return values
}

func removeTrailingEmptyStrings(slice []interface{}) []interface{} {
	n := len(slice)
	for i := n - 1; i >= 0; i-- {
		if str, ok := slice[i].(string); ok && strings.TrimSpace(str) != "" {
			return slice[:i+1]
		}
	}
	return nil
}

func removeEmptyRows(values [][]interface{}) [][]interface{} {
	n := len(values)
	for i := n - 1; i >= 0; i-- {
		isEmpty := true
		for _, item := range values[i] {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			values = values[:i]
		} else {
			break
		}
	}
	return values
}

func (m RangeResourceModel) ReplaceValues(originalValues [][]interface{}) [][]interface{} {
	// clear original
	result := [][]interface{}{}

	for i := range originalValues {
		result = append(result, []interface{}{})
		for range originalValues[i] {
			result[i] = append(result[i], "")
		}
	}

	newValues := m.RowsToValues()

	for i := range newValues {
		// if it is a new row
		if len(result) >= i {
			//append it
			result = append(result, []interface{}{})
		}
		for j := range newValues[i] {
			// if original contains the position
			if len(result) > i && len(result[i]) > j {
				//replace
				result[i][j] = newValues[i][j]
			} else {
				// append
				result[i] = append(result[i], newValues[i][j])
			}
		}
	}
	return result
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
	updateRequest.Context(ctx)
	updateRequest.ValueInputOption(data.ValueInputOption.ValueString())
	_, err := updateRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to perform update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState implements resource.ResourceWithImportState.
func (r *RangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data RangeResourceModel

	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("ID is not correct", "The ID must be a <spreadsheet_id>:<range>, but it was "+req.ID)
		return
	}

	data.SpreadsheetID = basetypes.NewStringValue(parts[0])
	data.Range = basetypes.NewStringValue(parts[1])

	getRequest := r.client.Spreadsheets.Values.Get(data.SpreadsheetID.ValueString(), data.Range.ValueString())
	getRequest.Context(ctx)
	getResponse, err := getRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data,", err.Error())
		return
	}

	data.Rows = ValuesToList(getResponse.Values)
	data.ValueInputOption = basetypes.NewStringValue("USER_ENTERED")

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
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

	getRequest := r.client.Spreadsheets.Values.Get(data.SpreadsheetID.ValueString(), data.Range.ValueString())
	getRequest.Context(ctx)
	getResponse, err := getRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data,", err.Error())
		return
	}

	data.Rows = ValuesToList(getResponse.Values)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
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
		Values: planData.ReplaceValues(stateData.RowsToValues()),
	})
	updateRequest.Context(ctx)
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
	var data RangeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	clearRequest := r.client.Spreadsheets.Values.Clear(data.SpreadsheetID.ValueString(), data.Range.ValueString(), &sheets.ClearValuesRequest{})
	clearRequest.Context(ctx)
	_, err := clearRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data,", err.Error())
		return
	}

}
