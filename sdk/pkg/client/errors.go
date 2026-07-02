package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error returned by the API
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Response   *http.Response
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// parseErrorResponse attempts to parse an error response from the API
func parseErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
			Response:   resp,
		}
	}

	// Try to parse as JSON error response
	var errorResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && (errorResp.Message != "" || errorResp.Reason != "") {
		message := errorResp.Message
		if message == "" {
			message = errorResp.Reason
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       errorResp.Code,
			Message:    message,
			Response:   resp,
		}
	}

	// Fallback to using status text and body
	message := string(body)
	if message == "" {
		message = resp.Status
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
		Response:   resp,
	}
}

// Error type checking helpers

// IsNotFound returns true if the error is a 404 Not Found error
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden error
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsBadRequest returns true if the error is a 400 Bad Request error
func IsBadRequest(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusBadRequest
	}
	return false
}

// IsConflict returns true if the error is a 409 Conflict error
func IsConflict(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusConflict
	}
	return false
}

// IsServerError returns true if the error is a 5xx server error
func IsServerError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}
	return false
}
