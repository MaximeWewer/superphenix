package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"
)

func TestBearerAuth(t *testing.T) {
	const secret = "gH7kP2vX9qL4mN8rT3wY6zB1cD5fJ0sA"

	tests := []struct {
		name       string
		configured string
		header     string
		wantStatus int
	}{
		{name: "valid bearer", configured: secret, header: "Bearer " + secret, wantStatus: http.StatusOK},
		{name: "wrong bearer", configured: secret, header: "Bearer " + secret[:len(secret)-1] + "X", wantStatus: http.StatusUnauthorized},
		{name: "prefix of the secret", configured: secret, header: "Bearer " + secret[:8], wantStatus: http.StatusUnauthorized},
		{name: "missing header", configured: secret, header: "", wantStatus: http.StatusUnauthorized},
		{name: "not a bearer", configured: secret, header: "Basic " + secret, wantStatus: http.StatusUnauthorized},
		{name: "empty configured secret never matches", configured: "", header: "Bearer ", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Http.AuthSecret = tt.configured
			handler := BearerAuth()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/org/project/instance", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
