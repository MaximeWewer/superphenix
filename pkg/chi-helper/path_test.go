package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRejectAmbiguousPath(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "plain path", target: "/org/project/instance/abc/start", wantStatus: http.StatusOK},
		{name: "query is not checked", target: "/org/project/instance?filter=a%2Fb", wantStatus: http.StatusOK},
		{name: "encoded slash", target: "/org/project/instance/a%2Fb/start", wantStatus: http.StatusBadRequest},
		{name: "encoded slash lowercase", target: "/org/project/instance/a%2fb/start", wantStatus: http.StatusBadRequest},
		{name: "encoded backslash", target: "/org/project/instance/a%5Cb/start", wantStatus: http.StatusBadRequest},
		{name: "dot dot segment", target: "/org/project/instance/../start", wantStatus: http.StatusBadRequest},
		{name: "encoded dot dot segment", target: "/org/project/instance/%2E%2E/start", wantStatus: http.StatusBadRequest},
		{name: "dot segment", target: "/org/project/./instance", wantStatus: http.StatusBadRequest},
		{name: "dots inside a segment", target: "/org/project/vm-type/a..b", wantStatus: http.StatusOK},
	}

	handler := RejectAmbiguousPath(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
