package openapi

// PaginationQueryParams returns the standard OpenAPI query parameters used by
// list endpoints that support core.ParsePagination.
//
// Supported query params:
//   - page  (1-based, integer)
//   - limit (items per page, integer)
func PaginationQueryParams() []Param {
	return []Param{
		{Name: "page", In: "query", Type: "integer"},
		{Name: "limit", In: "query", Type: "integer"},
	}
}
