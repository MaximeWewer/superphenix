package group

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/config"

	v1 "github.com/super-phenix/superphenix/pkg/permify-wrapper/pkg/base/v1"
	v1PSet "github.com/super-phenix/superphenix/pkg/permify-wrapper/pkg/base/v1/permissionSet"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCheckRequestedPermissionSets(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		current   []string
		wantErr   bool
	}{
		{name: "public sets", requested: []string{v1PSet.IAMReadOnly, v1PSet.ProjectDiskFullAccess}},
		{name: "member set is tolerated", requested: []string{v1PSet.SpxMember, v1PSet.BillingReadOnly}},
		{name: "empty", requested: nil},
		{name: "owner set", requested: []string{v1PSet.IAMReadOnly, v1PSet.SpxOwner}, wantErr: true},
		{name: "unknown internal set", requested: []string{"spx_admin"}, wantErr: true},
		{name: "unknown set", requested: []string{"NotAPermissionSet"}, wantErr: true},
		{name: "owner set kept on a legacy group", requested: []string{v1PSet.SpxOwner}, current: []string{v1PSet.SpxOwner, v1PSet.SpxMember}},
		{name: "owner set added to an existing group", requested: []string{v1PSet.SpxOwner}, current: []string{v1PSet.IAMReadOnly}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRequestedPermissionSets(tt.requested, tt.current)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAssignablePermissionSetsDropsOwnerFromPredefinedGroups(t *testing.T) {
	for _, predefined := range v1.PredefinedGroups {
		sets := assignablePermissionSets(predefined.Sets())
		assert.NotContains(t, sets, v1PSet.SpxOwner, "group %s", predefined.Key)
		assert.NotContains(t, sets, v1PSet.SpxMember, "group %s", predefined.Key)
		for _, ps := range sets {
			assert.True(t, isAssignablePermissionSet(ps), "group %s: %s", predefined.Key, ps)
		}
	}
}

func TestCreateGroupRefusesInternalPermissionSets(t *testing.T) {
	// No query is expected: the request must be refused before any database
	// or Permify access.
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	cfg := &config.Config{}
	cfg.PublicHTTP.MaxBodySize = 1
	svc := New(cfg)
	orgaId := uuid.New().String()

	for _, body := range []string{
		`{"name":"escalate","allProjects":true,"permissionSets":["spx_owner"]}`,
		`{"name":"unknown","permissionSets":["NotAPermissionSet"]}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/organization/"+orgaId+"/iam/group", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("orgaId", orgaId)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()
		svc.CreateOrUpdateOrganizationGroup(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code, body)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}
