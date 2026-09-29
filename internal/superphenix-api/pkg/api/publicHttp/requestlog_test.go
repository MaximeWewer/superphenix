package publicHttp

import (
	"bytes"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestRequestLoggerRedactsBearer(t *testing.T) {
	var buf bytes.Buffer
	logged := middleware.RequestLogger(redactingLogFormatter{
		LogFormatter: &middleware.DefaultLogFormatter{Logger: stdlog.New(&buf, "", 0), NoColor: true},
	})

	var seen string
	handler := logged(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query().Get("bearer")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/org/api/spx-ctrl/az1/p/instance/x/vnc?bearer=eyJsecret&x=1", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if strings.Contains(buf.String(), "eyJsecret") || !strings.Contains(buf.String(), "bearer=REDACTED") {
		t.Errorf("log line = %q", buf.String())
	}
	if seen != "eyJsecret" {
		t.Errorf("handler saw bearer %q, the request itself must not be modified", seen)
	}
}
