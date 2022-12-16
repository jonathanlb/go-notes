package notes

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func Test_CreateNotesDb(t *testing.T) {
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

func Test_SharesNote(t *testing.T) {
	dbFileName := ":memory:"
	tmpDirName := t.TempDir()
	dbFileName = tmpDirName + "/notes.sqlite3"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Second), DEFAULT_ACCESS, 0,
	}
	id, err := CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	assert.NotNil(t, id, "Unexpected nil on note creation")

	// Check that we can share notes.
	authorId2, err := CreateAuthor(db, "Another Test User", "")
	assert.Nil(t, err, "Unexpected error on author creation")
	SharesWith(db, 1, authorId2)

	retrievedNote, err := GetNote(db, authorId2, 1)
	assert.Nil(t, err, "Unexpected error on note sharing")
	assert.NotNil(t, retrievedNote, "Unexpected nil on note sharing")

	// Ensure that sharing is not the default.
	note = NoteRecord{
		authorId2, "#Some note", int(time.Second), DEFAULT_ACCESS, 0,
	}
	id, err = CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	retrievedNote, err = GetNote(db, 1, id)
	assert.Nil(t, err, "Unexpected error on unshared note retrieval")
	assert.Nil(t, retrievedNote, "Unexpected sharing of unshared note")

	// Ensure only author can read private notes.
	note = NoteRecord{
		1, "#Private note", int(time.Second), PRIVATE_ACCESS, 0,
	}
	id, err = CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on private note insertion")
	assert.NotNil(t, id, "Unexpected nil on private note creation")

	retrievedNote, err = GetNote(db, 1, id)
	assert.Nil(t, err, "Unexpected error on private note retrieval")
	assert.NotNil(t, retrievedNote, "Unexpected nil on private note retrieval")

	retrievedNote, err = GetNote(db, authorId2, id)
	assert.Nil(t, err, "Unexpected error on private note denied retrieval")
	assert.Nil(t, retrievedNote, "Unexpected private note sharing")
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
