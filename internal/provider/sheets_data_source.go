// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/api/sheets/v4"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RowsDataSource{}

func NewRowsDataSource() datasource.DataSource {
	return &RowsDataSource{}
}

// RowsDataSource defines the data source implementation.
type RowsDataSource struct {
	client *sheets.Service
}

// RowsDataSourceModel describes the data source data model.
type RowsDataSourceModel struct {
	SheetID types.String `tfsdk:"sheet_id"`
	Range   types.String `tfsdk:"range"`

	Rows types.List   `tfsdk:"rows"`
	Raw  types.String `tfsdk:"raw"`
}

func (d *RowsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rows"
}

func (d *RowsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Rows data source",

		Attributes: map[string]schema.Attribute{
			"sheet_id": schema.StringAttribute{
				MarkdownDescription: "The file to get the rows from",
				Required:            true,
			},
			"range": schema.StringAttribute{
				MarkdownDescription: "The range to read",
				Required:            true,
			},
			"rows": schema.ListAttribute{
				ElementType: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "The rows",
				Computed:            true,
			},
			"raw": schema.StringAttribute{
				MarkdownDescription: "the raw data",
				Computed:            true,
			},
		},
	}
}

func (d *RowsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RowsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RowsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	request := d.client.Spreadsheets.Values.Get(data.SheetID.ValueString(), data.Range.ValueString())
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

	data.Raw = types.StringValue(fmt.Sprint(values.Values))

	tfAttr := []attr.Value{}

	for _, row := range values.Values {
		tfRow := []attr.Value{}
		for _, el := range row {
			tfRow = append(tfRow, types.StringValue(el.(string)))
		}
		tfList := types.ListValueMust(types.StringType, tfRow)
		tfAttr = append(tfAttr, tfList)
	}

	data.Rows = types.ListValueMust(types.ListType{ElemType: types.StringType}, tfAttr)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
