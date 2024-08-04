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
					mux.HandleFunc("POST /v4/spreadsheets/{spreadsheetIdUpdate}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := strings.Split(r.PathValue("spreadsheetIdUpdate"), ":")[0]
						defer r.Body.Close()
						requestBody := &sheets.BatchUpdateSpreadsheetRequest{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

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
						err = json.NewEncoder(w).Encode(res)
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
}`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_sheet.test", "spreadsheet_id", "test-spreadsheet-id"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.title", "test title"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.index", "1"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.sheet_id", "2"),
				),
			},
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					mux.HandleFunc("POST /v4/spreadsheets/{spreadsheetIdUpdate}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := strings.Split(r.PathValue("spreadsheetIdUpdate"), ":")[0]
						defer r.Body.Close()
						requestBody := &sheets.BatchUpdateSpreadsheetRequest{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res := sheets.BatchUpdateSpreadsheetResponse{
							SpreadsheetId: spreadsheetID,
							Replies: []*sheets.Response{
								{
									AddSheet: &sheets.AddSheetResponse{
										Properties: &sheets.SheetProperties{Title: requestBody.Requests[0].UpdateSheetProperties.Properties.Title},
									},
								},
							},
						}
						err = json.NewEncoder(w).Encode(res)
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
		title = "test title change"
	}
}`, server.URL),

				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_sheet.test", "spreadsheet_id", "test-spreadsheet-id"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.title", "test title change"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.index", "1"),
					resource.TestCheckResourceAttr("gsheets_sheet.test", "properties.sheet_id", "2"),
				),
			},
		},
	})
}
