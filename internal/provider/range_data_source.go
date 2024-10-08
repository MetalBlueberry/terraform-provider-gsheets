package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/api/sheets/v4"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RangeDataSource{}

func NewRangeDataSource() datasource.DataSource {
	return &RangeDataSource{}
}

// RangeDataSource defines the data source implementation.
type RangeDataSource struct {
	client *sheets.Service
}

// RangeDataSourceModel describes the data source data model.
type RangeDataSourceModel struct {
	SpreadsheetID  types.String `tfsdk:"spreadsheet_id"`
	Range          types.String `tfsdk:"range"`
	Values         types.List   `tfsdk:"values"`
	MajorDimension types.String `tfsdk:"major_dimension"`
}

func (d *RangeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_range"
}

func (d *RangeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Allows to fetch data from a spreadsheet by providing the spreadsheet_id and the range.

To fetch data from a specific sheet, you must use the range syntax to point to a specific sheet.`,

		Attributes: map[string]schema.Attribute{
			"spreadsheet_id": schema.StringAttribute{
				MarkdownDescription: "The unique ID for the spreadsheet. It can be obtained from the URL.",
				Required:            true,
			},
			"range": schema.StringAttribute{
				MarkdownDescription: "The range to read. It follows standard range notation documented in google sheets.",
				Required:            true,
			},
			"values": schema.ListAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "The data that will be read",
				Computed:            true,
			},
			"major_dimension": schema.StringAttribute{
				MarkdownDescription: "major dimension for the values",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ROWS", "COLUMNS"),
				},
			},
		},
	}
}

func (d *RangeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sheets.Service)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *RangeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RangeDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.Spreadsheets.Values.Get(data.SpreadsheetID.ValueString(), data.Range.ValueString())
	if !data.MajorDimension.IsNull() {
		request.MajorDimension(data.MajorDimension.ValueString())
	}

	values, err := request.Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"unexpected error fetching data",
			err.Error(),
		)
		return
	}
	if values.HTTPStatusCode != 200 {
		resp.Diagnostics.AddError(
			"Unexpected status code",
			fmt.Sprintf("Received status %d", values.HTTPStatusCode),
		)
		return
	}

	data.Values = ValuesToList(values.Values)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func ValuesToList(values [][]interface{}) basetypes.ListValue {
	tfAttr := []attr.Value{}

	for _, row := range values {
		tfRow := []attr.Value{}
		for _, el := range row {
			if el == nil {
				tfRow = append(tfRow, types.StringValue(""))
			} else {
				v, ok := el.(string)
				if !ok {
					tfRow = append(tfRow, types.StringValue(""))
				} else {
					tfRow = append(tfRow, types.StringValue(v))
				}
			}
		}
		tfList := types.ListValueMust(types.StringType, tfRow)
		tfAttr = append(tfAttr, tfList)
	}

	return types.ListValueMust(types.ListType{ElemType: types.StringType}, tfAttr)
}
