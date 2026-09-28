package project

import (
	"regexp"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupMockDB creates a sqlmock-backed gorm.DB and swaps it into db.Client.
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
		sqlDB.Close()
	}
}

// TestFindByIdAndOrgaId verifies the authorization guard used before project
// deletion: the query must be scoped by BOTH the project id and the
// organization id, and a project that does not belong to the organization must
// yield an error (so RemoveProject aborts before any destructive action).
func TestFindByIdAndOrgaId(t *testing.T) {
	projectID := uuid.New()
	orgaID := uuid.New()

	t.Run("returns the project when it belongs to the organization", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{"id", "orga_id", "name"}).
			AddRow(projectID, orgaID, "my-project")
		// The WHERE clause must carry the orga_id filter, not just the id.
		mock.ExpectQuery(regexp.QuoteMeta(`"orga_id"`)).WillReturnRows(rows)

		p, err := FindByIdAndOrgaId(projectID, orgaID)
		assert.NoError(t, err)
		assert.Equal(t, projectID, p.ID)
		assert.Equal(t, orgaID, p.OrgaId)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error for a project of another organization", func(t *testing.T) {
		mock, cleanup := setupMockDB(t)
		defer cleanup()

		// Row set is empty: the project id exists but not under this orga_id.
		rows := sqlmock.NewRows([]string{"id", "orga_id", "name"})
		mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

		_, err := FindByIdAndOrgaId(projectID, uuid.New())
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
