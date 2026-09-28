package secret

import (
	"errors"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{name: "random secret", value: "gH7kP2vX9qL4mN8rT3wY6zB1cD5fJ0sA"},
		{name: "minimum length", value: "0123456789abcdef"},
		{name: "empty", value: "", wantErr: ErrEmpty},
		{name: "former chart default", value: "secret!", wantErr: ErrKnownValue},
		{name: "former code default", value: "secret", wantErr: ErrKnownValue},
		{name: "known default, other case", value: "ChangeMe", wantErr: ErrKnownValue},
		{name: "too short", value: "0123456789abcde", wantErr: ErrTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.value)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("Check() error = %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
