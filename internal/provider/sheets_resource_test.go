// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"google.golang.org/api/sheets/v4"
)

func TestAccSheetResource(t *testing.T) {
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
					mux.HandleFunc("POST /v4/spreadsheets", func(w http.ResponseWriter, r *http.Request) {

						defer r.Body.Close()
						requestBody := &sheets.Spreadsheet{}
						json.NewDecoder(r.Body).Decode(requestBody)

						res := sheets.Spreadsheet{
							Properties: &sheets.SpreadsheetProperties{
								Title: requestBody.Properties.Title,
							},
							SpreadsheetId: "test-sheet-id",
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
	range = "A:C"
	spreadsheet = {
		properties  = {
			title = "test document title"
		}
	}

}`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_sheet.test", "sheet_id", "test-sheet-id"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "spreadsheet.properties.title", "test document title"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "rows.#", "0"),
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
							UpdatedRange:  updateRange,
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
	spreadsheet = {
		properties  = {
			title = "test document title"
		}
	}
	range = "A:C"
	rows = [
				["a","b","c"],
				["1","2","3"],
	]

}`, server.URL),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_sheet.test", "sheet_id", "test-sheet-id"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "spreadsheet.properties.title", "test document title"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "rows.#", "2"),
				),
			},
		},
	})
}
