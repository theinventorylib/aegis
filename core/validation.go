package core

import (
	"encoding/json"
	"fmt"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// WriteValidationError writes a structured validation error response.
// It extracts field-level errors from ozzo-validation and returns them
// in a standardized format for API consumers.
func WriteValidationError(w http.ResponseWriter, err error) {
	if errs, ok := err.(validation.Errors); ok {
		validationErrors := make(map[string]string)
		for field, fieldErr := range errs {
			validationErrors[field] = fieldErr.Error()
		}

		WriteJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Error:   "Validation failed",
			Data: map[string]interface{}{
				"validation_errors": validationErrors,
			},
		})
		return
	}

	// Fallback for non-validation errors
	WriteJSON(w, http.StatusBadRequest, &Response{
		Success: false,
		Error:   err.Error(),
	})
}

// BindAndValidate decodes a JSON request body and validates it.
// T must implement a Validate() error method.
// This helper ensures consistent validation across all handlers.
//
// Example usage:
//
//	req, err := core.BindAndValidate[CreateOrganizationRequest](r)
//	if err != nil {
//	    core.WriteValidationError(w, err)
//	    return
//	}
func BindAndValidate[T interface{ Validate() error }](r *http.Request) (T, error) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := req.Validate(); err != nil {
		return req, err
	}
	return req, nil
}

// ValidateMiddleware creates a middleware that automatically validates request bodies.
// T must implement a Validate() error method.
// The validated request is passed to the handler, eliminating the need for manual validation.
//
// Example usage:
//
//	router.POST("/organizations", ValidateMiddleware(p.CreateOrganizationHandler))
//
//	func (p *Plugin) CreateOrganizationHandler(
//	    w http.ResponseWriter,
//	    r *http.Request,
//	    req CreateOrganizationRequest,  // Already validated!
//	) {
//	    // Use req directly - validation is guaranteed
//	}
func ValidateMiddleware[T interface{ Validate() error }](
	handler func(w http.ResponseWriter, r *http.Request, req T),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := BindAndValidate[T](r)
		if err != nil {
			WriteValidationError(w, err)
			return
		}
		handler(w, r, req)
	}
}
