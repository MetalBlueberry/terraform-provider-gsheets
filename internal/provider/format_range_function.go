package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the desired interfaces.
var _ function.Function = &FormatRangeFunction{}

type FormatRangeFunction struct{}

func NewFormatRangeFunction() function.Function {
	return &FormatRangeFunction{}
}

func (f *FormatRangeFunction) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "format_range"
}

func (f *FormatRangeFunction) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Helps to properly format google sheet ranges",
		Description: `Given a sheet reference and a range in any valid notation, it concatenates both values and adds the quotes

It will be equivalent to "'${gsheets_sheet.test.properties.title}'!A:C" but will look like format_range(gsheets_sheet,"A:C")
		`,
		Parameters: []function.Parameter{
			function.ObjectParameter{
				AttributeTypes: map[string]attr.Type{
					"properties": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"title": types.StringType,
						},
					},
				},
				Name:        "sheet",
				Description: "The sheet object to extract the name from",
			},
			function.StringParameter{
				Name:        "range",
				Description: "Range in any valid notation.",
			},
		},
		Return: function.StringReturn{},
	}
}

type FormatRangeSpreadsheetPropertiesModel struct {
	Title types.String `tfsdk:"title"`
}

type FormatRangeSheetsResourceModel struct {
	Properties *FormatRangeSpreadsheetPropertiesModel `tfsdk:"properties"`
}

func (f *FormatRangeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var sheetRange string
	var sheet FormatRangeSheetsResourceModel

	// Read Terraform argument data into the variables
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &sheet, &sheetRange))
	if resp.Error != nil {
		return
	}

	rangeResult := fmt.Sprintf("'%s'!%s", sheet.Properties.Title.ValueString(), sheetRange)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, rangeResult))
}
