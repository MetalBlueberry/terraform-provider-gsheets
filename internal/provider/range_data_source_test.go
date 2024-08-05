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

func TestAccRowsDataSource(t *testing.T) {
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
					mux.HandleFunc("/v4/spreadsheets/{sheetID}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						res := sheets.ValueRange{
							Values: [][]interface{}{
								{"a", "b", "c"},
								{"1", "2", "3"},
							},
							Range:          r.PathValue("range"),
							MajorDimension: "ROWS",
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

data "gsheets_range" "test" {
  spreadsheet_id = "example-sheet-id"
  range    = "A1:B10"
}
`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gsheets_range.test", "spreadsheet_id", "example-sheet-id"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "range", "A1:B10"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "values.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.gsheets_range.test", "values.0.*", "a"),
					resource.TestCheckTypeSetElemAttr("data.gsheets_range.test", "values.1.*", "3"),
				),
			},
			{

				PreConfig: func() {
					mux = http.NewServeMux()
					mux.HandleFunc("/v4/spreadsheets/{sheetID}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						majorDimension := r.URL.Query().Get("majorDimension")
						if majorDimension != "COLUMNS" {
							t.Errorf("Expected major dimension to be 'COLUMNS' but it was '%s'", majorDimension)
						}

						res := sheets.ValueRange{
							Values: [][]interface{}{
								{"a", "1"},
								{"b", "2"},
								{"c", "3"},
							},
							Range:          r.PathValue("range"),
							MajorDimension: majorDimension,
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

data "gsheets_range" "test" {
  spreadsheet_id = "example-sheet-id"
  range    = "A1:B10"
  major_dimension = "COLUMNS"
}
`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gsheets_range.test", "spreadsheet_id", "example-sheet-id"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "range", "A1:B10"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "values.#", "3"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "values.0.#", "2"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "values.1.#", "2"),
					resource.TestCheckResourceAttr("data.gsheets_range.test", "values.2.#", "2"),
				),
			},
		},
	})
}
