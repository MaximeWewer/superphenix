package netguard

import (
	"context"
	"errors"
	"net"
	"testing"
)

type fakeResolver map[string][]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(ip)})
	}
	return out, nil
}

func TestCheckURL(t *testing.T) {
	resolver := fakeResolver{
		"images.example.org":  {"93.184.216.34"},
		"dual.example.org":    {"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"},
		"rebind.example.org":  {"93.184.216.34", "10.0.0.5"},
		"private.example.org": {"192.168.1.10"},
		"mirror.corp":         {"10.1.2.3"},
	}
	opts := Options{Schemes: []string{"http", "https", "docker"}, Resolver: resolver}

	tests := []struct {
		name    string
		url     string
		opts    *Options
		wantErr bool
	}{
		{name: "public https", url: "https://images.example.org/disk.qcow2"},
		{name: "public registry", url: "docker://images.example.org/library/alpine:3"},
		{name: "public dual stack", url: "http://dual.example.org:8080/img"},
		{name: "public IP literal", url: "https://93.184.216.34/img"},
		{name: "scheme not allowed", url: "ftp://images.example.org/img", wantErr: true},
		{name: "file scheme", url: "file:///etc/passwd", wantErr: true},
		{name: "credentials", url: "https://user:pass@images.example.org/img", wantErr: true},
		{name: "no host", url: "https:///img", wantErr: true},
		{name: "private IPv4 literal", url: "http://10.0.0.1/img", wantErr: true},
		{name: "loopback", url: "http://127.0.0.1/img", wantErr: true},
		{name: "link-local metadata address", url: "http://169.254.169.254/latest", wantErr: true},
		{name: "CGNAT", url: "http://100.64.0.1/img", wantErr: true},
		{name: "load balancer VIP range", url: "http://198.18.0.10/img", wantErr: true},
		{name: "IPv4-mapped private", url: "http://[::ffff:10.0.0.1]/img", wantErr: true},
		{name: "IPv6 loopback", url: "http://[::1]/img", wantErr: true},
		{name: "IPv6 unique local", url: "http://[fd00::1]/img", wantErr: true},
		{name: "single label name", url: "http://registry/img", wantErr: true},
		{name: "localhost", url: "http://localhost:8080/img", wantErr: true},
		{name: "cluster service name", url: "http://api.default.svc/img", wantErr: true},
		{name: "cluster FQDN", url: "http://api.default.svc.cluster.local./img", wantErr: true},
		{name: "resolves to private", url: "https://private.example.org/img", wantErr: true},
		{name: "one private address among public", url: "https://rebind.example.org/img", wantErr: true},
		{name: "unresolvable host", url: "https://unknown.example.org/img", wantErr: true},
		{name: "allowed internal mirror", url: "https://mirror.corp/img", opts: &Options{Schemes: []string{"https"}, AllowedHosts: []string{"mirror.corp"}, Resolver: resolver}},
		{name: "denied public range", url: "https://images.example.org/img", opts: &Options{Schemes: []string{"https"}, DeniedCIDRs: []string{"93.184.216.0/24"}, Resolver: resolver}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := opts
			if tt.opts != nil {
				o = *tt.opts
			}
			err := CheckURL(context.Background(), tt.url, o)
			if tt.wantErr != (err != nil) {
				t.Errorf("CheckURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
