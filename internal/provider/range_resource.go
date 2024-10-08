package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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
	Values           types.List   `tfsdk:"values"`
	MajorDimension   types.String `tfsdk:"major_dimension"`
}

func (m RangeResourceModel) ToInterface() [][]interface{} {
	values := [][]interface{}{}
	for _, el := range m.Values.Elements() {
		row := []interface{}{}
		elListValue, _ := el.(basetypes.ListValue)
		for _, ell := range elListValue.Elements() {
			ellString, _ := ell.(types.String)
			row = append(row, ellString.ValueString())
		}
		values = append(values, row)
	}
	return values
}

// I need to write unit test for the reason why I have this :harold:.
func (m RangeResourceModel) ToCleanInterface() [][]interface{} {
	return Clean(m.ToInterface())
}

// Clean removes empty strings at the end of rows and empty rows.
// It mimics what gogole sheets does when fetching data from empty cells.
func Clean(slice [][]interface{}) [][]interface{} {
	for i := range slice {
		slice[i] = removeTrailingEmptyStrings(slice[i])
	}
	slice = removeEmptyRows(slice)

	return slice
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

// Clear replaces all values for empty strings.
func Clear(reference [][]interface{}) [][]interface{} {
	result := [][]interface{}{}

	for i := range reference {
		result = append(result, []interface{}{})
		for range reference[i] {
			result[i] = append(result[i], "")
		}
	}
	return result
}

// Merge takes values on b and replaces them on a.
// It leaves non matching elements as they are on a.
func Merge(a, b [][]interface{}) [][]interface{} {
	for i := range b {
		// if it is a new row
		if len(a) == i {
			//append it
			a = append(a, []interface{}{})
		}
		for j := range b[i] {
			// if original contains the position
			if len(a) > i && len(a[i]) > j {
				//replace
				a[i][j] = b[i][j]
			} else {
				// append
				a[i] = append(a[i], b[i][j])
			}
		}
	}
	return a
}

// KeepDimensions uses reference to get the desired dimensions and data for the values. If data is bigger, the dimensions will grow. Otherwise, it will fit the reference dimensions.
func KeepDimensions(reference [][]interface{}, data [][]interface{}) [][]interface{} {
	result := Clear(reference)
	return Merge(result, data)
}

func (m RangeResourceModel) KeepDimensions(reference [][]interface{}) [][]interface{} {
	newValues := m.ToInterface()
	return KeepDimensions(reference, newValues)
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"values": schema.ListAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "The rows",
				Optional:            true,
				Computed:            true,
				Default: listdefault.StaticValue(basetypes.NewListValueMust(types.ListType{
					ElemType: types.StringType,
				}, []attr.Value{})),
			},
			"major_dimension": schema.StringAttribute{
				MarkdownDescription: "major dimension for the values",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ROWS", "COLUMNS"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := r.buildUpdateCall(ctx, &data)
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
	if !data.MajorDimension.IsNull() {
		getRequest.MajorDimension(data.MajorDimension.ValueString())
	}

	getRequest.Context(ctx)
	getResponse, err := getRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data,", err.Error())
		return
	}

	data.Values = ValuesToList(getResponse.Values)
	data.ValueInputOption = basetypes.NewStringValue("USER_ENTERED")

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Read is called when the provider must read resource values in order
// to update state. Planned state values should be read from the
// ReadRequest and new state values set on the ReadResponse.
func (r *RangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RangeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getRequest := r.client.Spreadsheets.Values.Get(data.SpreadsheetID.ValueString(), data.Range.ValueString())
	if !data.MajorDimension.IsNull() {
		getRequest.MajorDimension(data.MajorDimension.ValueString())
	}

	getRequest.Context(ctx)
	getResponse, err := getRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read data,", err.Error())
		return
	}

	rowValues := data.ToInterface()
	extended := KeepDimensions(rowValues, getResponse.Values)
	data.Values = ValuesToList(extended)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Update is called to update the state of the resource. Config, planned
// state, and prior state values should be read from the
// UpdateRequest and new state values set on the UpdateResponse.
func (r *RangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var newState RangeResourceModel
	var originalState RangeResourceModel
	var planData RangeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &newState)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &originalState)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newState.Values = planData.Values
	newState.ValueInputOption = planData.ValueInputOption

	planData.Values = ValuesToList(KeepDimensions(originalState.ToInterface(), planData.ToInterface()))
	updateRequest := r.buildUpdateCall(ctx, &planData)
	_, err := updateRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to perform update request", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete is called when the provider must delete the resource. Config
// values may be read from the DeleteRequest.
//
// If execution completes without error, the framework will automatically
// call DeleteResponse.State.RemoveResource(), so it can be omitted
// from provider logic.
func (r *RangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RangeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.Values = ValuesToList(Clear(data.ToInterface()))

	clearRequest := r.buildUpdateCall(ctx, &data)
	clearRequest.Context(ctx)
	_, err := clearRequest.Do()
	if err != nil {
		resp.Diagnostics.AddError("Unable to update data,", err.Error())
		return
	}

}

func (r *RangeResource) buildUpdateCall(ctx context.Context, data *RangeResourceModel) *sheets.SpreadsheetsValuesUpdateCall {
	updateBody := &sheets.ValueRange{
		Range:  data.Range.ValueString(),
		Values: data.ToInterface(),
	}
	if !data.MajorDimension.IsNull() {
		updateBody.MajorDimension = data.MajorDimension.ValueString()
	}

	updateRequest := r.client.Spreadsheets.Values.Update(data.SpreadsheetID.ValueString(), data.Range.ValueString(), updateBody)
	updateRequest.Context(ctx)
	updateRequest.ValueInputOption(data.ValueInputOption.ValueString())
	return updateRequest
}
