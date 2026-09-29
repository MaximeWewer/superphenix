package nat

import "testing"

func TestValidateDNAT(t *testing.T) {
	tests := []struct {
		name               string
		external, internal string
		protocol           string
		wantErr            bool
	}{
		{name: "tcp", external: "80", internal: "8080", protocol: "tcp"},
		{name: "udp uppercase", external: "53", internal: "53", protocol: "UDP"},
		{name: "default protocol", external: "443", internal: "443", protocol: ""},
		{name: "port bounds", external: "1", internal: "65535", protocol: "tcp"},
		{name: "surrounding spaces", external: " 22 ", internal: "22", protocol: " tcp "},
		{name: "ranges", external: "30000-30010", internal: "30000-30010", protocol: "tcp"},
		{name: "empty port", external: "", internal: "80", protocol: "tcp", wantErr: true},
		{name: "zero port", external: "0", internal: "80", protocol: "tcp", wantErr: true},
		{name: "port too high", external: "65536", internal: "80", protocol: "tcp", wantErr: true},
		{name: "signed port", external: "+80", internal: "80", protocol: "tcp", wantErr: true},
		{name: "non numeric port", external: "80a", internal: "80", protocol: "tcp", wantErr: true},
		{name: "port with space inside", external: "80 81", internal: "80", protocol: "tcp", wantErr: true},
		{name: "reversed range", external: "90-80", internal: "80", protocol: "tcp", wantErr: true},
		{name: "open range", external: "80-", internal: "80", protocol: "tcp", wantErr: true},
		{name: "double range", external: "1-2-3", internal: "80", protocol: "tcp", wantErr: true},
		{name: "invalid internal port", external: "80", internal: "-1", protocol: "tcp", wantErr: true},
		{name: "unknown protocol", external: "80", internal: "80", protocol: "icmp", wantErr: true},
		{name: "protocol with extra text", external: "80", internal: "80", protocol: "tcp x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNAT(tt.external, tt.internal, tt.protocol)
			if tt.wantErr != (err != nil) {
				t.Errorf("ValidateDNAT(%q, %q, %q) error = %v, wantErr %v", tt.external, tt.internal, tt.protocol, err, tt.wantErr)
			}
		})
	}
}
