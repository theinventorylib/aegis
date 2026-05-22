package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TC-HTTP-001: ParsePagination with valid values
func TestParsePagination_ValidValues(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedPage   int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "page 2, limit 50",
			query:          "?page=2&limit=50",
			expectedPage:   2,
			expectedLimit:  50,
			expectedOffset: 50,
		},
		{
			name:           "page 1, limit 1",
			query:          "?page=1&limit=1",
			expectedPage:   1,
			expectedLimit:   1,
			expectedOffset:  0,
		},
		{
			name:           "page 10, limit 100 (max)",
			query:          "?page=10&limit=100",
			expectedPage:   10,
			expectedLimit:  100,
			expectedOffset: 900,
		},
		{
			name:           "no query params (defaults)",
			query:          "",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			url := "/organizations" + tt.query
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// When
			result := ParsePagination(req)

			// Then
			if result.Page != tt.expectedPage {
				t.Errorf("Page = %d, want %d", result.Page, tt.expectedPage)
			}
			if result.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", result.Limit, tt.expectedLimit)
			}
			if result.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", result.Offset, tt.expectedOffset)
			}
		})
	}
}

// TC-HTTP-002: ParsePagination with invalid values
func TestParsePagination_InvalidValues(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedPage   int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "negative page defaults to 1",
			query:          "?page=-1&limit=10",
			expectedPage:   1,
			expectedLimit:  10,
			expectedOffset:  0,
		},
		{
			name:           "page 0 defaults to 1",
			query:          "?page=0&limit=10",
			expectedPage:   1,
			expectedLimit:  10,
			expectedOffset:  0,
		},
		{
			name:           "limit over max defaults to 20",
			query:          "?page=1&limit=200",
			expectedPage:    1,
			expectedLimit:   20,
			expectedOffset:  0,
		},
		{
			name:           "limit 0 defaults to 20",
			query:          "?page=1&limit=0",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "negative limit defaults to 20",
			query:          "?page=1&limit=-5",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "non-numeric page defaults",
			query:          "?page=abc&limit=10",
			expectedPage:   1,
			expectedLimit:  10,
			expectedOffset:  0,
		},
		{
			name:           "non-numeric limit defaults",
			query:          "?page=1&limit=abc",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "both invalid",
			query:          "?page=abc&limit=xyz",
			expectedPage:   1,
			expectedLimit:  20,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			url := "/organizations" + tt.query
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// When
			result := ParsePagination(req)

			// Then
			if result.Page != tt.expectedPage {
				t.Errorf("Page = %d, want %d", result.Page, tt.expectedPage)
			}
			if result.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", result.Limit, tt.expectedLimit)
			}
			if result.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", result.Offset, tt.expectedOffset)
			}
		})
	}
}

// TC-HTTP-003: ParsePagination offset calculation
func TestParsePagination_OffsetCalculation(t *testing.T) {
	tests := []struct {
		name           string
		page           string
		limit          string
		expectedOffset int
	}{
		{
			name:           "page 1 offset 0",
			page:           "1",
			limit:          "20",
			expectedOffset: 0,
		},
		{
			name:           "page 2 offset 20",
			page:           "2",
			limit:          "20",
			expectedOffset: 20,
		},
		{
			name:           "page 3 offset 40",
			page:           "3",
			limit:          "20",
			expectedOffset: 40,
		},
		{
			name:           "page 5 limit 10 offset 40",
			page:           "5",
			limit:          "10",
			expectedOffset: 40,
		},
		{
			name:           "page 1 limit 50 offset 0",
			page:           "1",
			limit:          "50",
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/test?page=%s&limit=%s", tt.page, tt.limit)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			result := ParsePagination(req)
			if result.Offset != tt.expectedOffset {
				t.Errorf("Offset = %d, want %d", result.Offset, tt.expectedOffset)
			}
		})
	}
}