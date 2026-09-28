// Package secret validates shared secrets loaded from the configuration.
package secret

import (
	"errors"
	"fmt"
	"strings"
)

// MinLength is the minimum length accepted for a shared secret.
const MinLength = 16

var (
	ErrEmpty      = errors.New("secret is empty")
	ErrKnownValue = errors.New("secret is a known default value")
	ErrTooShort   = fmt.Errorf("secret is shorter than %d characters", MinLength)
	knownDefaults = []string{"secret", "secret!", "changeme", "changeme!", "change-me", "password", "admin"}
)

// Check returns an error when a shared secret is empty, too short or one of
// the default values that ship (or used to ship) with the charts and configs.
// Services must refuse to start with such a secret: anyone able to reach them
// would otherwise authenticate with a publicly known value.
func Check(value string) error {
	if value == "" {
		return ErrEmpty
	}
	for _, known := range knownDefaults {
		if strings.EqualFold(value, known) {
			return ErrKnownValue
		}
	}
	if len(value) < MinLength {
		return ErrTooShort
	}
	return nil
}
