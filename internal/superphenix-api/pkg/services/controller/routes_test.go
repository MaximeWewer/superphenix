package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	oldClient := db.Client
	db.Client = gormDB
	return mock, func() {
		db.Client = oldClient
		_ = sqlDB.Close()
	}
}

func serveWithParams(params map[string]string) (*httptest.ResponseRecorder, bool) {
	called := false
	handler := CheckProjectInOrganization(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec, called
}

var projectLookup = regexp.QuoteMeta(`SELECT * FROM "projects" WHERE (id = $1 AND orga_id = $2)`)

func TestCheckProjectInOrganization(t *testing.T) {
	orgaId := uuid.New()
	projectId := uuid.New()

	t.Run("project of the organization", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()
		mock.ExpectQuery(projectLookup).
			WithArgs(projectId, orgaId, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "orga_id"}).AddRow(projectId, orgaId))

		rec, called := serveWithParams(map[string]string{"orgaId": orgaId.String(), "projectId": projectId.String()})
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("project of another organization", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()
		mock.ExpectQuery(projectLookup).
			WithArgs(projectId, orgaId, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "orga_id"}))

		rec, called := serveWithParams(map[string]string{"orgaId": orgaId.String(), "projectId": projectId.String()})
		assert.False(t, called)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid ids are refused without a query", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()

		for _, params := range []map[string]string{
			{"orgaId": orgaId.String(), "projectId": "not-a-uuid"},
			{"orgaId": "not-a-uuid", "projectId": projectId.String()},
			{"orgaId": orgaId.String(), "projectId": uuid.Nil.String()},
		} {
			rec, called := serveWithParams(params)
			assert.False(t, called)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("route without project", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()

		rec, called := serveWithParams(map[string]string{"orgaId": orgaId.String()})
		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSharedMiddlewaresCheckProjectInOrganization(t *testing.T) {
	want := reflect.ValueOf(CheckProjectInOrganization).Pointer()
	for _, mw := range SharedMiddlewares() {
		if reflect.ValueOf(mw).Pointer() == want {
			return
		}
	}
	t.Fatal("CheckProjectInOrganization is not in the shared controller middlewares")
}
