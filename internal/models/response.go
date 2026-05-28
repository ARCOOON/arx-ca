package models

// APIResponse is the standard envelope for all API JSON responses.
type APIResponse struct {
	Error *string `json:"error"`
	Data  any     `json:"data"`
}

// NewSuccessResponse builds a successful API response with the given payload.
func NewSuccessResponse(data any) APIResponse {
	return APIResponse{
		Error: nil,
		Data:  data,
	}
}

// NewErrorResponse builds a failed API response with a sanitized client message.
func NewErrorResponse(message string) APIResponse {
	return APIResponse{
		Error: &message,
		Data:  nil,
	}
}
