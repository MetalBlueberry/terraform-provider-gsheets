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
