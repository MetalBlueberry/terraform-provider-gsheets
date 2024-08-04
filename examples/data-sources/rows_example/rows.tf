
data "gsheets_rows" "test" {
  // The id can be obtained from the browser URL
  spreadsheet_id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  range          = "A1:B10"
}
