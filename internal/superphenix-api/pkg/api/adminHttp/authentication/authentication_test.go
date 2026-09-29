package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecretAuth(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		presented  string
		want       int
	}{
		{name: "valid secret", configured: "s3cret-value", presented: "s3cret-value", want: http.StatusOK},
		{name: "wrong secret", configured: "s3cret-value", presented: "s3cret-valuX", want: http.StatusUnauthorized},
		{name: "prefix of the secret", configured: "s3cret-value", presented: "s3cret", want: http.StatusUnauthorized},
		{name: "missing secret", configured: "s3cret-value", presented: "", want: http.StatusUnauthorized},
		{name: "empty configured secret", configured: "", presented: "", want: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := SecretAuth(tt.configured)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.presented != "" {
				req.Header.Set("Secret", tt.presented)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
