package trmlabs

import "errors"

var (
	// ErrAPIUnavailable is returned when TRM Labs API is unavailable
	ErrAPIUnavailable = errors.New("TRM Labs API unavailable")

	// ErrInvalidAddress is returned when the address format is invalid
	ErrInvalidAddress = errors.New("invalid wallet address")

	// ErrInvalidChain is returned when the chain is not supported
	ErrInvalidChain = errors.New("invalid or unsupported chain")

	// ErrRateLimitExceeded is returned when API rate limit is exceeded
	ErrRateLimitExceeded = errors.New("TRM Labs API rate limit exceeded")

	// ErrUnauthorized is returned when API key is invalid
	ErrUnauthorized = errors.New("TRM Labs API unauthorized")
)
