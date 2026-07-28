package tests

import (
	"database/sql"
	"github.com/daywee-zx/go-photo-library/pkg/storage"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testDBPath     = "test_images/test.db"
	testImageBatch = 9
)

func setupTestDB(t *testing.T) *storage.Storage {
	db, err := sql.Open("sqlite", "file:testdb?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	store := storage.NewStorage(db)
	require.NoError(t, store.Init())

	t.Cleanup(func() {
		store.Close()
	})
	return store
}
