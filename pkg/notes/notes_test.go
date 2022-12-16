package notes

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func Test_CreateNotesDb(t *testing.T) {
	// tmpDirName := t.TempDir()
	// dbFileName := tmpDirName + "/notes.sqlite3"
	dbFileName := ":memory:"

	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Second), DEFAULT_ACCESS, 0,
	}
	id, err := CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")

	retrievedNote, err := GetNote(db, 1, id)
	assert.Nil(t, err, "Unexpected error on note retrieval")
	assert.NotNil(t, retrievedNote, "Unexpected nil on note retrieval")
	assert.Equal(t, note.Content, retrievedNote.Content, "Retrieving note content")
}

func Test_trimContentToTitle(t *testing.T) {
	trimmed := trimContentToTitle("#Title goes here\nblah, blah")
	assert.Equal(t, "Title goes here", trimmed, "Strip out MD")
}

func createDb(dbFileName string) (*sql.DB, error) {
	db, err := CreateNoteDb(dbFileName)
	if err != nil || db == nil {
		return nil, err
	}
	_, err = CreateAuthor(db, "Test User", "")
	if err != nil || db == nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
