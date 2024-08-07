
data "gsheets_range" "test" {
  // The id can be obtained from the browser URL
  spreadsheet_id = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  range          = "A1:B10"
}


output "rows" {
  value = gsheets_range.test.values
}
