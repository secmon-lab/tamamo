package scenario

// AuthStrategy defines how a login endpoint handles authentication attempts.
// When attached to a route, the server tracks attempts per source IP
// and returns failure responses until the required number of failures is reached.
type AuthStrategy struct {
	// FailuresBeforeSuccess is the number of failed attempts before allowing success.
	FailuresBeforeSuccess int `json:"failures_before_success"`

	// FailureStatusCode is the HTTP status code for failed attempts (e.g., 401, 403).
	FailureStatusCode int `json:"failure_status_code"`

	// FailureBody is the response body for failed attempts.
	FailureBody string `json:"failure_body"`

	// FailureHeaders are additional headers for failed responses.
	FailureHeaders map[string]string `json:"failure_headers,omitempty"`
}
