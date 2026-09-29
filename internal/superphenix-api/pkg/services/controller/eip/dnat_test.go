package eip

import "testing"

func TestValidateDNATRules(t *testing.T) {
	valid := EipDNATBody{ExternalPort: "443", InternalIP: "10.0.0.1", InternalPort: "8443", Protocol: "tcp"}
	tests := []struct {
		name    string
		rules   []EipDNATBody
		wantErr bool
	}{
		{name: "no rule"},
		{name: "valid rule", rules: []EipDNATBody{valid}},
		{name: "invalid external port", rules: []EipDNATBody{valid, {ExternalPort: "443,22", InternalIP: "10.0.0.1", InternalPort: "22", Protocol: "tcp"}}, wantErr: true},
		{name: "invalid internal port", rules: []EipDNATBody{{ExternalPort: "443", InternalIP: "10.0.0.1", InternalPort: "0", Protocol: "tcp"}}, wantErr: true},
		{name: "invalid protocol", rules: []EipDNATBody{{ExternalPort: "443", InternalIP: "10.0.0.1", InternalPort: "443", Protocol: "all"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateDNATRules(tt.rules); tt.wantErr != (err != nil) {
				t.Errorf("validateDNATRules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
