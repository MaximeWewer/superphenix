package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	ch "github.com/super-phenix/superphenix/pkg/chi-helper"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// newControllerStub mimics the controller router: CleanPath, then a project
// scoped route that echoes the project it resolved.
func newControllerStub(t *testing.T) *httptest.Server {
	t.Helper()
	ctrl := chi.NewRouter()
	ctrl.Use(middleware.CleanPath)
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, chi.URLParam(r, "projectId"))
	}
	ctrl.Get("/{orgId}/{projectId}/instance/{effectiveId}/start", handler)
	ctrl.Get("/{orgId}/{projectId}/instance", handler)
	return httptest.NewServer(ctrl)
}

// newAPIStub mimics an API route forwarded with SimpleRedirect.
func newAPIStub(t *testing.T, controllerURL string, withPathCheck bool) *httptest.Server {
	t.Helper()
	api := chi.NewRouter()
	if withPathCheck {
		api.Use(ch.RejectAmbiguousPath)
	}
	api.Use(middleware.CleanPath)
	api.Get("/api/spx-ctrl/{az}/{orgId}/{projectId}/instance/{effectiveId}/start", func(w http.ResponseWriter, r *http.Request) {
		p, err := ReverseProxy(controllerURL, "/api/spx-ctrl/az1", "")
		if err != nil {
			t.Fatal(err)
		}
		p.ServeHTTP(w, r)
	})
	return httptest.NewServer(api)
}

const traversalTarget = "/api/spx-ctrl/az1/org/mine/instance/x%2F..%2F..%2F..%2Fother%2Finstance%2Fvm/start"

func TestProxyEncodedSeparatorsStayInProject(t *testing.T) {
	ctrl := newControllerStub(t)
	defer ctrl.Close()

	for _, withPathCheck := range []bool{true, false} {
		api := newAPIStub(t, ctrl.URL, withPathCheck)

		resp, err := http.Get(api.URL + traversalTarget)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		api.Close()

		if withPathCheck && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("with path check: status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
		// Even without the API check, the proxy must not let the controller
		// resolve the encoded separators into another project.
		if string(body) == "other" {
			t.Errorf("withPathCheck=%v: controller resolved project %q", withPathCheck, body)
		}
	}
}

func TestProxyForwardsPlainPath(t *testing.T) {
	ctrl := newControllerStub(t)
	defer ctrl.Close()
	api := newAPIStub(t, ctrl.URL, true)
	defer api.Close()

	resp, err := http.Get(api.URL + "/api/spx-ctrl/az1/org/mine/instance/abc/start")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK || string(body) != "mine" {
		t.Errorf("status = %d, body = %q, want 200 and %q", resp.StatusCode, body, "mine")
	}
}
