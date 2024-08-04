// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
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
					resource.TestCheckResourceAttr("gsheets_range.test_range", "rows.#", "0"),
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
