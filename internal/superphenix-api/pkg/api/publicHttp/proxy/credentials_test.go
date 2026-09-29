package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
)

func TestReverseProxyDropsUserCredentials(t *testing.T) {
	p, err := ReverseProxy("http://controller:8080", "/api/spx-ctrl/az1", "controller-secret")
	if err != nil {
		t.Fatal(err)
	}

	in := httptest.NewRequest(http.MethodGet, "/api/spx-ctrl/az1/org/p/instance/x/vnc?bearer=user-token&keep=1", nil)
	in.Header.Set("Cookie", "ory_kratos_session=abc; refresh=def")
	out := in.Clone(in.Context())
	pr := &httputil.ProxyRequest{In: in, Out: out}
	p.Rewrite(pr)

	if got := pr.Out.URL.Query().Get("bearer"); got != "" {
		t.Errorf("bearer forwarded: %q", got)
	}
	if got := pr.Out.URL.Query().Get("keep"); got != "1" {
		t.Errorf("other query params must be kept, got %q", got)
	}
	if got := pr.Out.Header.Get("Cookie"); got != "" {
		t.Errorf("cookies forwarded: %q", got)
	}
	if got := pr.Out.Header.Get("Authorization"); got != "Bearer controller-secret" {
		t.Errorf("Authorization = %q, want the controller secret", got)
	}
}

func TestRewriteRequestDropsUserCredentials(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/spx-ctrl/az1/org/p/instance?bearer=user-token&page=2", nil)
	r.Header.Set("Cookie", "ory_kratos_session=abc")
	if err := RewriteRequest(r, "http://controller:8080", "/api/spx-ctrl/az1"); err != nil {
		t.Fatal(err)
	}
	if r.URL.Query().Get("bearer") != "" || r.URL.Query().Get("page") != "2" {
		t.Errorf("query = %q", r.URL.RawQuery)
	}
	if r.Header.Get("Cookie") != "" {
		t.Error("cookies forwarded")
	}
}
