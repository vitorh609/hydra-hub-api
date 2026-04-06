package clickup

import "errors"

var (
	ErrConnectionNotFound  = errors.New("clickup connection not found")
	ErrIntegrationDisabled = errors.New("clickup integration disabled")
	ErrInvalidCredentials  = errors.New("clickup invalid credentials")
	ErrUpstreamUnavailable = errors.New("clickup upstream unavailable")
	ErrBadGateway          = errors.New("clickup bad gateway")
)
