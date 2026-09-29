package kaas

import (
	"context"
	"errors"
	"net"
	"testing"
)

type fakeRepoResolver map[string]string

func (f fakeRepoResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ip, ok := f[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil
}

// Tests of this package never depend on real DNS.
func init() {
	postInstallRepoResolver = fakeRepoResolver{
		"kubernetes.github.io": "185.199.108.153",
		"charts.example.org":   "93.184.216.34",
		"charts.corp":          "10.0.0.8",
	}
}

func TestCheckPostInstallRepoHost(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "public https repository", url: "https://kubernetes.github.io/ingress-nginx"},
		{name: "public oci repository", url: "oci://charts.example.org/charts"},
		{name: "private IP", url: "https://10.0.0.1/charts", wantErr: true},
		{name: "link-local IP", url: "http://169.254.169.254/charts", wantErr: true},
		{name: "cluster service", url: "http://chartmuseum.tools.svc.cluster.local/charts", wantErr: true},
		{name: "single label host", url: "http://chartmuseum/charts", wantErr: true},
		{name: "host resolving to a private IP", url: "https://charts.corp/charts", wantErr: true},
		{name: "unresolvable host", url: "https://unknown.example.org/charts", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPostInstallRepoHost(context.Background(), tt.url)
			if tt.wantErr != (err != nil) {
				t.Errorf("checkPostInstallRepoHost(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
