package tests

import (
	"context"
	"testing"

	"github.com/daywee-zx/go-photo-library/pkg/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertEntry(t *testing.T) {
	store := setupTestDB(t)

	emptyEmbedding := make([]float32, 1024)
	testCases := []struct {
		name    string
		entry   storage.IndexedEntry
		wantErr bool
	}{
		{
			"valid-tags",
			storage.IndexedEntry{
				Entry: storage.Entry{
					ID:   0,
					Path: "test_images/test1.jpg",
					Tags: []string{"soldier", "russian"},
				},
				VisualEmbed: emptyEmbedding,
				TextEmbed:   emptyEmbedding,
			},
			false,
		},
		{
			"valid-notags",
			storage.IndexedEntry{
				Entry: storage.Entry{
					ID:   0,
					Path: "test_images/test2.jpg",
					Tags: []string{},
				},
				VisualEmbed: emptyEmbedding,
				TextEmbed:   emptyEmbedding,
			},
			false,
		},
		{
			"nonvalid-tags",
			storage.IndexedEntry{
				Entry: storage.Entry{
					ID:   0,
					Path: "",
					Tags: []string{"soldier", "russian"},
				},
				VisualEmbed: emptyEmbedding,
				TextEmbed:   emptyEmbedding,
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := store.InsertEntry(tc.entry)
			if tc.wantErr {
				require.Error(t, err, "expected an error")
				return
			}
			require.NoError(t, err, "error not expected")

			entry, err := store.GetEntry(context.Background(), id)
			require.NoError(t, err, "error not expected")
			assert.ElementsMatch(t, tc.entry.Tags, entry.Tags, "tags do not match")
		})
	}
}

func TestDeleteEntry(t *testing.T) {
	store := setupTestDB(t)

	emptyEmbedding := make([]float32, 1024)
	entries := []storage.IndexedEntry{
		{
			Entry: storage.Entry{
				ID:   0,
				Path: "test_images/test1.jpg",
				Tags: []string{"soldier", "russian"},
			},
			VisualEmbed: emptyEmbedding,
			TextEmbed:   emptyEmbedding,
		},
		{
			Entry: storage.Entry{
				ID:   0,
				Path: "test_images/test2.jpg",
				Tags: []string{},
			},
			VisualEmbed: emptyEmbedding,
			TextEmbed:   emptyEmbedding,
		},
		{
			Entry: storage.Entry{
				ID:   0,
				Path: "test_images/test3.jpg",
				Tags: []string{"soldier", "english"},
			},
			VisualEmbed: emptyEmbedding,
			TextEmbed:   emptyEmbedding,
		},
	}
	var ids []int64
	for _, e := range entries {
		id, err := store.InsertEntry(e)
		require.NoError(t, err, "error not expected while inserting")
		ids = append(ids, id)
	}

	ctx := context.Background()

	// delete no tags
	err := store.DeleteEntry(ids[1])
	require.NoError(t, err)
	entry, err := store.GetEntry(ctx, ids[1])
	assert.Equal(t, "", entry.Path, "entry was not deleted")

	// delete with tags
	err = store.DeleteEntry(ids[0])
	require.NoError(t, err)
	entry, err = store.GetEntry(ctx, ids[0])
	assert.Equal(t, "", entry.Path, "entry was not deleted")

	// ensure others' tags were not cascade deleted
	tags, err := store.GetTags(ids[2])
	assert.Len(t, tags, 2, "entry has unsufficient tags")

	// delete remaining
	err = store.DeleteEntry(ids[2])
	require.NoError(t, err)
	entry, err = store.GetEntry(ctx, ids[2])
	assert.Equal(t, "", entry.Path, "entry was not deleted")
}
