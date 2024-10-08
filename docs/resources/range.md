---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gsheets_range Resource - gsheets"
subcategory: ""
description: |-
  Sheets resource
---

# gsheets_range (Resource)

Sheets resource

## Example Usage

```terraform
resource "gsheets_sheet" "test" {
  // The id can be obtained from the browser URL
  spreadsheet_id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  properties = {
    title = "test title"
  }
}
resource "gsheets_range" "test_range" {
  spreadsheet_id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  range          = provider::gsheets::format_range(gsheets_sheet.test, "A:C")
  values = [
    ["a", "b", "c"],
    [1, 2, 3],
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `range` (String) The range to read
- `spreadsheet_id` (String) The file to get the rows from

### Optional

- `major_dimension` (String) major dimension for the values
- `value_input_option` (String) how to post data
- `values` (List of List of String) The rows
