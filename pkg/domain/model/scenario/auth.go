package scenario

// AuthStrategy defines how a login endpoint handles authentication attempts.
// When attached to a route, the server tracks unique credential submissions per source IP.
// The first MinFailures unique credentials always fail. After that, each new unique
// credential succeeds with SuccessProbability chance (determined once and cached).
type AuthStrategy struct {
	// MinFailures is the minimum number of unique credentials that must fail
	// before any credential can succeed.
	MinFailures int `json:"min_failures"`

	// SuccessProbability is the probability (0.0-1.0) that a new unique credential
	// succeeds after MinFailures is reached. A value of 0.3 means ~30% chance.
	SuccessProbability float64 `json:"success_probability"`

	// FailureStatusCode is the HTTP status code for failed attempts (e.g., 401, 403).
	FailureStatusCode int `json:"failure_status_code"`

	// FailureBody is the response body for failed attempts.
	FailureBody string `json:"failure_body"`

	// FailureHeaders are additional headers for failed responses.
	FailureHeaders map[string]string `json:"failure_headers,omitempty"`

	// CredentialFields specifies which request body fields are considered credentials.
	// Only these fields are used to determine unique login attempts.
	// If empty, the entire request body is used as a fallback.
	CredentialFields []string `json:"credential_fields,omitempty"`
}
