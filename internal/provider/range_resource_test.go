package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
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
					//Expect create
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
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {

						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
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
					mux.HandleFunc("GET /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						updateRange := r.PathValue("range")

						res := sheets.ValueRange{
							Range:  updateRange,
							Values: [][]interface{}{},
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
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "0"),
				),
			},
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					var storedValues [][]interface{}
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						storedValues = requestBody.Values

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
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
					mux.HandleFunc("GET /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						updateRange := r.PathValue("range")

						res := sheets.ValueRange{
							Range:  updateRange,
							Values: storedValues,
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
	values = [
				["a","b","c"],
				[1,2,3],
	]
}
	`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", "'test title'!A:C"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "2"),
				),
			},
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					var storedValues [][]interface{}

					// I know this is a dirty hack, but I don't want to import external libraries now.
					updateCalls := 0
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						updateCalls++

						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						if updateCalls > 1 {
							if requestBody.MajorDimension != "COLUMNS" {
								t.Errorf("Expected major dimension 'COLUMNS' but got '%s'", requestBody.MajorDimension)
							}
						}

						storedValues = requestBody.Values

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
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
					getCallCount := 0
					mux.HandleFunc("GET /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						getCallCount++
						updateRange := r.PathValue("range")

						if getCallCount == 2 {
							if majorDimension := r.URL.Query().Get("majorDimension"); majorDimension != "COLUMNS" {
								t.Errorf("Expected major dimension 'COLUMNS' but got '%s'", majorDimension)
							}
						}

						res := sheets.ValueRange{
							Range:  updateRange,
							Values: storedValues,
						}
						err := json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
					mux.HandleFunc("POST /v4/spreadsheets/{spreadsheetId}/values/{rangeclear}", func(w http.ResponseWriter, r *http.Request) {
						updateRange := strings.Split(r.PathValue("rangeclear"), ":")[0]
						spreadsheetID := r.PathValue("spreadsheetId")
						defer r.Body.Close()

						requestBody := &sheets.ClearValuesRequest{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res := sheets.ClearValuesResponse{
							ClearedRange:  updateRange,
							SpreadsheetId: spreadsheetID,
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
}

resource "gsheets_range" "test_range" {
	spreadsheet_id = "test-spreadsheet-id"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	major_dimension = "COLUMNS"
	values = [
				["a","1"],
				["b","2"],
				["c","3"],
	]
}
	`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", "'test title'!A:C"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "3"),
				),
			},
			{
				PreConfig: func() {
					mux = http.NewServeMux()
					var storedValues [][]interface{}
					mux.HandleFunc("PUT /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						spreadsheetID := r.PathValue("spreadsheetId")
						updateRange := r.PathValue("range")
						defer r.Body.Close()
						requestBody := &sheets.ValueRange{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						if requestBody.MajorDimension != "COLUMNS" {
							t.Errorf("Expected major dimension 'COLUMNS' but got '%s'", requestBody.MajorDimension)
						}

						storedValues = Clean(requestBody.Values)

						res := sheets.UpdateValuesResponse{
							SpreadsheetId: spreadsheetID,
							UpdatedData: &sheets.ValueRange{
								Range:  updateRange,
								Values: requestBody.Values,
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
					mux.HandleFunc("GET /v4/spreadsheets/{spreadsheetId}/values/{range}", func(w http.ResponseWriter, r *http.Request) {
						updateRange := r.PathValue("range")

						if majorDimension := r.URL.Query().Get("majorDimension"); majorDimension != "COLUMNS" {
							t.Errorf("Expected major dimension 'COLUMNS' but got '%s'", majorDimension)
						}

						res := sheets.ValueRange{
							Range:  updateRange,
							Values: storedValues,
						}
						err := json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
					mux.HandleFunc("POST /v4/spreadsheets/{spreadsheetId}/values/{rangeclear}", func(w http.ResponseWriter, r *http.Request) {
						updateRange := strings.Split(r.PathValue("rangeclear"), ":")[0]
						spreadsheetID := r.PathValue("spreadsheetId")
						defer r.Body.Close()

						requestBody := &sheets.ClearValuesRequest{}
						err := json.NewDecoder(r.Body).Decode(requestBody)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						res := sheets.ClearValuesResponse{
							ClearedRange:  updateRange,
							SpreadsheetId: spreadsheetID,
						}
						err = json.NewEncoder(w).Encode(res)
						if err != nil {
							log.Println(err)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(200)
					})
					// Expect delete
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
								{},
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
}

resource "gsheets_range" "test_range" {
	spreadsheet_id = "test-spreadsheet-id"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	major_dimension = "COLUMNS"
	values = [
				["a","1"],
				["b","2"],
	]
}
	`, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", "'test title'!A:C"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "2"),
				),
			},
		},
	})
}

// Relies on the existence of a document that the service account has access to.
func TestIntegrationRangeResource_RowChanges(t *testing.T) {
	configVars := config.Variables{
		"service_account_credentials": config.StringVariable(os.Getenv("SERVICE_ACCOUNT_CREDENTIALS")),
	}

	expectedTitle := "test-title"
	if v := os.Getenv("SHEET_TITLE"); v != "" {
		expectedTitle = v
	}
	configVars["sheet_title"] = config.StringVariable(expectedTitle)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ConfigVariables: configVars,
				Config: `
variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}

variable "sheet_title" {
  description = "sheet title for the test run"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	properties = {
		title = var.sheet_title
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
}
	`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", fmt.Sprintf("'%s'!A:C", expectedTitle)),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "0"),
				),
			},
			{
				ConfigVariables: configVars,
				Config: `
variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}
variable "sheet_title" {
  description = "sheet title for the test run"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	properties = {
		title = var.sheet_title
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	values = [
	["a","b","c"],
	[1,2,3],
	]
}
	`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", fmt.Sprintf("'%s'!A:C", expectedTitle)),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "2"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.0.#", "3"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.1.#", "3"),
				),
			},
			{
				ConfigVariables: configVars,
				Config: `
variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}
variable "sheet_title" {
  description = "sheet title for the test run"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	properties = {
		title = var.sheet_title
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	values = [
	["a","b","c"],
	[1,2,""],
	["x",2,"z"],
	["","","zx"],
	]
}
	`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", fmt.Sprintf("'%s'!A:C", expectedTitle)),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "4"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.0.#", "3"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.1.#", "3"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.2.#", "3"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.3.#", "3"),
				),
			},
			{
				ConfigVariables: configVars,
				Config: `
variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}
variable "sheet_title" {
  description = "sheet title for the test run"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	properties = {
		title = var.sheet_title
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	range = "'${gsheets_sheet.test.properties.title}'!A:C"
	values = [
	["",""],
	]
}
	`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", fmt.Sprintf("'%s'!A:C", expectedTitle)),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "1"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.0.#", "2"),
				),
			},
			{
				ConfigVariables: configVars,
				Config: `
variable "service_account_credentials" {
  description = "json value with the token obtained from the console"
  type        = string
}
variable "sheet_title" {
  description = "sheet title for the test run"
  type        = string
}

provider "gsheets" {
  service_account_key = var.service_account_credentials
}

resource "gsheets_sheet" "test" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	properties = {
		title = var.sheet_title
	}
}
resource "gsheets_range" "test_range" {
	spreadsheet_id = "1gk-q5dVEvkkdxno0FPwxAZo8_KsCDicW_MAs0KQAF8w"
	range = "'${gsheets_sheet.test.properties.title}'!D:F"
	values = [
	["a","b","c"],
	[1,2,3],
	]
}
	`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gsheets_range.test_range", "range", fmt.Sprintf("'%s'!D:F", expectedTitle)),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.#", "2"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.0.#", "3"),
					resource.TestCheckResourceAttr("gsheets_range.test_range", "values.1.#", "3"),
				),
			},
		},
	})
}

func TestRemoveTrailingEmptyStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name:     "No trailing empty strings",
			input:    []interface{}{"first", "second", "third"},
			expected: []interface{}{"first", "second", "third"},
		},
		{
			name:     "Trailing empty strings",
			input:    []interface{}{"first", "second", "", "", "third"},
			expected: []interface{}{"first", "second", "", "", "third"},
		},
		{
			name:     "All elements are empty",
			input:    []interface{}{"", "", "", ""},
			expected: []interface{}{},
		},
		{
			name:     "Mixed elements",
			input:    []interface{}{"first", "", "second", "", "", "third", ""},
			expected: []interface{}{"first", "", "second", "", "", "third"},
		},
		{
			name:     "Empty input",
			input:    []interface{}{},
			expected: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeTrailingEmptyStrings(tt.input)
			if !equal(result, tt.expected) {
				t.Errorf("removeTrailingEmptyStrings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to check equality of slices.
func equal(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRemoveEmptyRows tests the removeEmptyRows function.
func TestRemoveEmptyRows(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]interface{}
		expected [][]interface{}
	}{
		{
			name:     "No empty rows",
			input:    [][]interface{}{{"first"}, {"second"}, {"third"}},
			expected: [][]interface{}{{"first"}, {"second"}, {"third"}},
		},
		{
			name:     "Trailing empty rows",
			input:    [][]interface{}{{"first"}, {"second"}, {""}, {""}},
			expected: [][]interface{}{{"first"}, {"second"}},
		},
		{
			name:     "All rows empty",
			input:    [][]interface{}{{""}, {""}},
			expected: [][]interface{}{},
		},
		{
			name:     "Mixed rows",
			input:    [][]interface{}{{"first"}, {""}, {"second"}, {""}, {""}},
			expected: [][]interface{}{{"first"}, {""}, {"second"}},
		},
		{
			name:     "Empty input",
			input:    [][]interface{}{},
			expected: [][]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeEmptyRows(tt.input)
			if !equal2D(result, tt.expected) {
				t.Errorf("removeEmptyRows() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to check equality of 2D slices.
func equal2D(a, b [][]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestKeepDimensions(t *testing.T) {
	tests := []struct {
		name      string
		reference [][]interface{}
		data      [][]interface{}
		expected  [][]interface{}
	}{
		{
			name:      "Basic Functionality",
			reference: [][]interface{}{{"", ""}, {"", ""}},
			data:      [][]interface{}{{1, 2}, {3, 4}},
			expected:  [][]interface{}{{1, 2}, {3, 4}},
		},
		{
			name:      "Different Dimensions - More Data Rows",
			reference: [][]interface{}{{"", ""}, {"", ""}},
			data:      [][]interface{}{{1, 2}, {3, 4}, {5, 6}},
			expected:  [][]interface{}{{1, 2}, {3, 4}, {5, 6}},
		},
		{
			name:      "Different Dimensions - More Data Columns",
			reference: [][]interface{}{{"", ""}, {"", ""}},
			data:      [][]interface{}{{1, 2, 3}, {4, 5, 6}},
			expected:  [][]interface{}{{1, 2, 3}, {4, 5, 6}},
		},
		{
			name:      "Empty Reference",
			reference: [][]interface{}{},
			data:      [][]interface{}{{1, 2}, {3, 4}},
			expected:  [][]interface{}{{1, 2}, {3, 4}},
		},
		{
			name:      "Empty Data",
			reference: [][]interface{}{{"", ""}, {"", ""}},
			data:      [][]interface{}{},
			expected:  [][]interface{}{{"", ""}, {"", ""}},
		},
		{
			name:      "Non-Uniform Dimensions",
			reference: [][]interface{}{{""}, {"", ""}, {"", "", ""}},
			data:      [][]interface{}{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
			expected:  [][]interface{}{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		},
		{
			name:      "Longer reference rows",
			reference: [][]interface{}{{"", "", ""}, {"", "", ""}},
			data:      [][]interface{}{{1, 2}, {3, 4}},
			expected:  [][]interface{}{{1, 2, ""}, {3, 4, ""}},
		},
		{
			name:      "more reference rows",
			reference: [][]interface{}{{"", "", ""}, {"", "", ""}, {"", "", ""}},
			data:      [][]interface{}{{1, 2}, {3, 4}},
			expected:  [][]interface{}{{1, 2, ""}, {3, 4, ""}, {"", "", ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := KeepDimensions(tt.reference, tt.data)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
