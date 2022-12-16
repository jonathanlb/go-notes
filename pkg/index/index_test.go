package index

import (
	"database/sql"
	"org/bredin/go-notes/pkg/notes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IndexNotesDb(t *testing.T) {
	tmpDirName := t.TempDir() // doesn't delete cleanly
	dbFileName := tmpDirName + "/notes.sqlite3"
	indexDirName := tmpDirName + "/test_index"

	db, _ := notes.CreateNoteDb(dbFileName)
	defer func(db *sql.DB) {
		db.Close()
	}(db)
	authorId := 1
	contents := []string{
		"hello", "ciao", "buonasera",
	}
	for _, content := range contents {
		if err := notes.CreateNote(db, &notes.NoteRecord{
			Author: authorId, Content: content, Created: 0, Privacy: notes.DEFAULT_ACCESS, RenderHint: 1,
		}); err != nil {
			t.Fatalf("Cannot insert note %s", err)
		}
	}

	index, err := CreateIndex(dbFileName, indexDirName)
	if err != nil || index == nil {
		t.Fatalf("Cannot create index %s", err)
	}

	docCount, _ := index.DocCount()
	assert.Equal(t, uint64(3), docCount, "Indexed all notes")

	searchResult, _ := SearchIndex(&index, "ciao")
	assert.Equal(t, 1, len(searchResult), "Expected only one relevant document")
}
