package log

import (
	"net/url"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	u, _ := url.Parse("https://api.example.org/org/api/spx-ctrl/az1/p/instance/x/vnc?bearer=eyJsecret&other=1")
	got := RedactURL(u)
	if strings.Contains(got, "eyJsecret") || !strings.Contains(got, "bearer=REDACTED") || !strings.Contains(got, "other=1") {
		t.Errorf("RedactURL() = %q", got)
	}
	if u.Query().Get("bearer") != "eyJsecret" {
		t.Error("RedactURL() must not modify the URL")
	}

	plain, _ := url.Parse("https://api.example.org/v1/session?next=/home")
	if got := RedactURL(plain); got != plain.String() {
		t.Errorf("RedactURL() = %q, want unchanged %q", got, plain.String())
	}
	if RedactURL(nil) != "" {
		t.Error("RedactURL(nil) must be empty")
	}
}

func TestStripSensitiveQuery(t *testing.T) {
	u, _ := url.Parse("http://controller/org/p/instance/x/vnc?bearer=eyJsecret&token=t&keep=1")
	StripSensitiveQuery(u)
	if u.RawQuery != "keep=1" {
		t.Errorf("RawQuery = %q, want %q", u.RawQuery, "keep=1")
	}
}
