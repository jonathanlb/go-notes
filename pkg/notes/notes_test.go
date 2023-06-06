package notes

import (
	"database/sql"
	"fmt"
	"reflect"
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
		1, "# My first note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
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
	// tmpDirName := t.TempDir()
	// dbFileName = tmpDirName + "/notes.sqlite3"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
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
		authorId2, "#Some note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
	}
	id, err = CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	retrievedNote, err = GetNote(db, 1, id)
	assert.NotNil(t, err, "Expected error on unshared note retrieval")
	assert.Nil(t, retrievedNote, "Unexpected sharing of unshared note")

	// Ensure only author can read private notes.
	note = NoteRecord{
		1, "#Private note", int(time.Now().Unix()), PRIVATE_ACCESS, 0,
	}
	id, err = CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on private note insertion")
	assert.NotNil(t, id, "Unexpected nil on private note creation")

	retrievedNote, err = GetNote(db, 1, id)
	assert.Nil(t, err, "Unexpected error on private note retrieval")
	assert.NotNil(t, retrievedNote, "Unexpected nil on private note retrieval")

	retrievedNote, err = GetNote(db, authorId2, id)
	assert.NotNil(t, err, "Expected error on private note denied retrieval")
	assert.Nil(t, retrievedNote, "Unexpected private note sharing")
}

func Test_GetsRecentNotes(t *testing.T) {
	dbFileName := ":memory:"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	userId := 1
	for i := 1; i < 6; i++ {
		note := NoteRecord{
			userId, "some note", i, DEFAULT_ACCESS, 0,
		}
		_, _ = CreateNote(db, &note)
	}

	recent, err := GetRecentNotes(db, userId, 2)
	assert.Nil(t, err, "Unexpected error getting recent notes")
	assert.Equal(t, 2, len(recent))
	expectedRecent := []int{5, 4}
	assert.True(t, reflect.DeepEqual(expectedRecent, recent),
		fmt.Sprintf("expected: %v, got %v", expectedRecent, recent))
}

func Test_ChecksPrivacyMode(t *testing.T) {
	dbFileName := ":memory:"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
	}
	id, err := CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	assert.NotNil(t, id, "Unexpected nil on note creation")

	err = SetNotePrivacy(db, 1, id, -1)
	assert.NotNil(t, err, "Expected error on set privacy with illegal mode")
	err = SetNotePrivacy(db, 1, id, 10)
	assert.NotNil(t, err, "Expected error on set privacy with illegal mode")
}

func Test_GuardsPrivacyUpdateById(t *testing.T) {
	dbFileName := ":memory:"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
	}
	id, err := CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	assert.NotNil(t, id, "Unexpected nil on note creation")

	err = SetNotePrivacy(db, 2, id, PRIVATE_ACCESS)
	assert.NotNil(t, err, "Expected error with unauthorized privacy update")
}

func Test_UpdatesPrivacy(t *testing.T) {
	dbFileName := ":memory:"
	db, err := createDb(dbFileName)
	assert.Nil(t, err, "Unexpected error on DB creation")
	defer db.Close()

	note := NoteRecord{
		1, "# My first note", int(time.Now().Unix()), DEFAULT_ACCESS, 0,
	}
	id, err := CreateNote(db, &note)
	assert.Nil(t, err, "Unexpected error on note insertion")
	assert.NotNil(t, id, "Unexpected nil on note creation")

	err = SetNotePrivacy(db, 1, id, PRIVATE_ACCESS)
	assert.Nil(t, err, "Unexpected error on set privacy")
	authorId2, err := CreateAuthor(db, "Another Test User", "")
	assert.Nil(t, err, "Unexpected error on author creation")
	SharesWith(db, 1, authorId2)

	retrievedNote, err := GetNote(db, authorId2, 1)
	assert.Nil(t, retrievedNote, "Expected note-sharing failure")
	assert.NotNil(t, err, "Expected error on private-note retrieval")
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
