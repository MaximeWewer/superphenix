package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	"github.com/go-chi/chi/v5"
)

func TestCheckBodyMatchesPath(t *testing.T) {
	const org = "11111111-1111-1111-1111-111111111111"
	const project = "22222222-2222-2222-2222-222222222222"
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgId", org)
	rctx.URLParams.Add("projectId", project)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	prefixed := spxId.FrameworkPrefix() + "-"
	tests := []struct {
		name    string
		m       spxId.Metadata
		wantErr bool
	}{
		{name: "same ids", m: spxId.Metadata{OrgId: org, ProjectId: project}},
		{name: "same ids with prefix", m: spxId.Metadata{OrgId: prefixed + org, ProjectId: prefixed + project}},
		{name: "other project", m: spxId.Metadata{OrgId: org, ProjectId: "33333333-3333-3333-3333-333333333333"}, wantErr: true},
		{name: "other organization", m: spxId.Metadata{OrgId: "44444444-4444-4444-4444-444444444444", ProjectId: project}, wantErr: true},
		{name: "empty body", m: spxId.Metadata{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckBodyMatchesPath(req, tt.m); tt.wantErr != (err != nil) {
				t.Errorf("CheckBodyMatchesPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
