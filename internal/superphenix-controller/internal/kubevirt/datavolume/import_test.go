package datavolume

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type fakeResolver map[string]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ip, ok := f[host]
	if !ok {
		return nil, errors.New("no such host")
	}
	return []net.IPAddr{{IP: net.ParseIP(ip)}}, nil
}

func useFakeResolver(t *testing.T) {
	t.Helper()
	old := importResolver
	importResolver = fakeResolver{
		"images.example.org": "93.184.216.34",
		"mirror.corp":        "10.1.2.3",
	}
	t.Cleanup(func() { importResolver = old })
}

func diskInfo(sourceType, url string) *CreateDiskInfo {
	info := &CreateDiskInfo{}
	info.OrgId = "11111111-1111-1111-1111-111111111111"
	info.ProjectId = "22222222-2222-2222-2222-222222222222"
	info.ResourceLocalId = "33333333-3333-3333-3333-333333333333"
	info.General.Storage = "10"
	info.General.Source = Source{Type: sourceType, URL: url}
	return info
}

func TestCreateDiskRefusesNonPublicImportURLs(t *testing.T) {
	useFakeResolver(t)

	tests := []struct {
		name       string
		sourceType string
		url        string
	}{
		{name: "http to a private address", sourceType: SourceTypeHttp, url: "http://10.0.0.1/disk.img"},
		{name: "http to the metadata address", sourceType: SourceTypeHttp, url: "http://169.254.169.254/latest"},
		{name: "http to a cluster service", sourceType: SourceTypeHttp, url: "http://minio.storage.svc:9000/disk.img"},
		{name: "http with an unsupported scheme", sourceType: SourceTypeHttp, url: "file:///etc/hosts"},
		{name: "registry on an internal name", sourceType: SourceTypeRegistry, url: "docker://registry:5000/img"},
		{name: "registry resolving to a private address", sourceType: SourceTypeRegistry, url: "docker://mirror.corp/img"},
		{name: "registry without docker scheme", sourceType: SourceTypeRegistry, url: "https://images.example.org/img"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := diskInfo(tt.sourceType, tt.url).CreateDisk(context.Background(), "spx-22222222-2222-2222-2222-222222222222")
			if !apierrors.IsBadRequest(err) {
				t.Fatalf("CreateDisk() error = %v, want BadRequest", err)
			}
		})
	}
}

func TestCheckImportURL(t *testing.T) {
	useFakeResolver(t)
	old := config.Global.ProductsConfig.ImportSources
	t.Cleanup(func() { config.Global.ProductsConfig.ImportSources = old })

	if err := checkImportURL(context.Background(), "https://images.example.org/disk.qcow2", "http", "https"); err != nil {
		t.Errorf("public URL refused: %v", err)
	}
	if err := checkImportURL(context.Background(), "docker://images.example.org/library/alpine:3", "docker"); err != nil {
		t.Errorf("public registry refused: %v", err)
	}

	config.Global.ProductsConfig.ImportSources.AllowedHosts = []string{"mirror.corp"}
	if err := checkImportURL(context.Background(), "docker://mirror.corp/img", "docker"); err != nil {
		t.Errorf("allowed internal mirror refused: %v", err)
	}

	config.Global.ProductsConfig.ImportSources.AllowedHosts = nil
	config.Global.ProductsConfig.ImportSources.DeniedCIDRs = []string{"93.184.216.0/24"}
	if err := checkImportURL(context.Background(), "https://images.example.org/disk.qcow2", "https"); err == nil {
		t.Error("URL in a denied range accepted")
	}
}
