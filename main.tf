terraform {
  required_providers {
    gsheets = {
      source = "MetalBlueberry/google-sheets"
    }
  }
}

provider "gsheets" {
  service_account_key = "credentials.prod.json"
}

resource "gsheets_sheet" "test_sheet" {
  spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
  properties = {
    title = "test_sheet"
  }
}


resource "gsheets_range" "test_range" {
  spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
  range          = "'${gsheets_sheet.test_sheet.properties.title}'!A:C"
  rows = [
    ["a ", "b", "c", ],
    ["1", "2", "3", ],
    ["", "4", "", ],
  ]
}
