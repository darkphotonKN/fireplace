package constants

import "errors"

var (
	ErrAuthentication     = errors.New("authentication error")
	ErrRateLimit          = errors.New("rate limit exceeded")
	ErrContextLength      = errors.New("context length exceeded")
	ErrServiceUnavailable = errors.New("service unavailable")
)