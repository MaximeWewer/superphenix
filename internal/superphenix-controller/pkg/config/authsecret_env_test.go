package config

import (
	"testing"
)

// TestAuthSecretFromEnv ensures the shared secret can be injected from a
// Kubernetes Secret through the environment, as the chart does, even though
// the default value is empty.
func TestAuthSecretFromEnv(t *testing.T) {
	const want = "gH7kP2vX9qL4mN8rT3wY6zB1cD5fJ0sA"
	t.Setenv("SUPERPHENIX-CONTROLLER_HTTP_AUTHSECRET", want)

	// No config file in the test directory: defaults plus environment only.
	if err := LoadConfig(); err != nil && err != FileNotFound {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if Global.Http.AuthSecret != want {
		t.Fatalf("Http.AuthSecret = %q, want the value from the environment", Global.Http.AuthSecret)
	}
}
