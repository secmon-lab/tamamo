package errutil

import "github.com/m-mizutani/goerr/v2"

var (
	// TagValidation indicates input validation failures.
	TagValidation = goerr.NewTag("validation")
	// TagNotFound indicates a requested resource was not found.
	TagNotFound = goerr.NewTag("not_found")
	// TagExternal indicates failures in LLM or external service calls.
	TagExternal = goerr.NewTag("external")
	// TagGeneration indicates failures during scenario generation.
	TagGeneration = goerr.NewTag("generation")
	// TagInternal indicates unexpected internal errors.
	TagInternal = goerr.NewTag("internal")
)
