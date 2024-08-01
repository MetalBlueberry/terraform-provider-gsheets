// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"google.golang.org/api/sheets/v4"
)

func TestAccRowsDataSource(t *testing.T) {
	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	testAccRowsDataSourceConfig := fmt.Sprintf(`
provider "gsheets" {
	endpoint = "%s"
}

data "gsheets_rows" "test" {
  sheet_id = "example-sheet-id"
  range    = "A1:B10"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			mux = http.NewServeMux()
			mux.HandleFunc("/v4/spreadsheets/{sheetID}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
				res := sheets.ValueRange{
					Values: [][]interface{}{
						{"a", "b", "c"},
						{"1", "2", "3"},
					},
					Range:          r.PathValue("range"),
					MajorDimension: "ROWS",
				}
				json.NewEncoder(w).Encode(res)
				w.WriteHeader(200)
			})
		},

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		IsUnitTest:               true,
		Steps: []resource.TestStep{
			{

				Config: testAccRowsDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gsheets_rows.test", "sheet_id", "example-sheet-id"),
					resource.TestCheckResourceAttr("data.gsheets_rows.test", "range", "A1:B10"),
					resource.TestCheckResourceAttr("data.gsheets_rows.test", "rows.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.gsheets_rows.test", "rows.0.*", "a"),
					resource.TestCheckTypeSetElemAttr("data.gsheets_rows.test", "rows.1.*", "3"),
				),
			},
		},
	})
}
