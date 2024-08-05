
resource "gsheets_sheet" "test" {
  // The id can be obtained from the browser URL
  spreadsheet_id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  properties = {
    title = "test title"
  }
}
