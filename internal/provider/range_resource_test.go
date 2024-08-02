// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"google.golang.org/api/sheets/v4"
)

func TestAccRangeResource(t *testing.T) {
	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		IsUnitTest:               true,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					mux.HandleFunc("POST /v4/spreadsheets/{spreadsheetIdUpdate}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := strings.Split(r.PathValue("spreadsheetIdUpdate"), ":")[0]
						defer r.Body.Close()
						requestBody := &sheets.BatchUpdateSpreadsheetRequest{}
						json.NewDecoder(r.Body).Decode(requestBody)

						res := sheets.BatchUpdateSpreadsheetResponse{
							SpreadsheetId: spreadsheetID,
							Replies: []*sheets.Response{
								{
									AddSheet: &sheets.AddSheetResponse{
										Properties: &sheets.SheetProperties{
											Index:   1,
											SheetId: 2,
											Title:   requestBody.Requests[0].AddSheet.Properties.Title,
										},
									},
								},
							},
						}
						err := json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						json.NewDecoder(r.Body).Decode(requestBody)

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
							},
						}
						err := json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
				},
				Config: fmt.Sprintf(`
provider "gsheets" {
	endpoint = "%s"
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "test-spreadsheet-id"
	properties = {
		title = "test title"
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "test-spreadsheet-id"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
}
	`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", "'test title'!A:C"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "rows.#", "0"),
				),
			},
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						json.NewDecoder(r.Body).Decode(requestBody)

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
							},
						}
						err := json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
				},
				Config: fmt.Sprintf(`
provider "gsheets" {
	endpoint = "%s"
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "test-spreadsheet-id"
	properties = {
		title = "test title"
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "test-spreadsheet-id"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	rows = [
				["a","b","c"],
				[1,2,3],
	]
}
	`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", "'test title'!A:C"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "rows.#", "2"),
				),
			},
		},
	})
}
