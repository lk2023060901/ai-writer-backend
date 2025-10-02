package types

import (
	"errors"
	"fmt"
)

var (
	// Configuration errors
	ErrInvalidProviderID        = errors.New("invalid provider ID")
	ErrInvalidProviderName      = errors.New("invalid provider name")
	ErrInvalidAPIHost           = errors.New("invalid API host")
	ErrMissingAPIKey            = errors.New("missing API key")
	ErrMissingBasicAuthPassword = errors.New("missing basic auth password")

	// Request errors
	ErrInvalidQuery = errors.New("invalid search query")
	ErrEmptyQuery   = errors.New("empty search query")
	ErrQueryTooLong = errors.New("query too long")

	// Provider errors
	ErrProviderNotFound     = errors.New("provider not found")
	ErrProviderNotAvailable = errors.New("provider not available")
	ErrProviderRateLimited  = errors.New("provider rate limited")
	ErrProviderUnauthorized = errors.New("provider unauthorized")
	ErrProviderTimeout      = errors.New("provider timeout")

	// Response errors
	ErrNoResults       = errors.New("no results found")
	ErrInvalidResponse = errors.New("invalid response from provider")
)

// ProviderError wraps provider-specific errors
type ProviderError struct {
	Provider ProviderID
	Code     string
	Message  string
	Err      error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s (%v)", e.Provider, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
