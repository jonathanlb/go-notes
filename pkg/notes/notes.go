package notes

import (
	"database/sql"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const PRIVATE_ACCESS = 0
const PROTECTED_ACCESS = 1
const PUBLIC_ACCESS = 2
const DEFAULT_ACCESS = 1

type NoteRecord struct {
	Author     int
	Content    string
	Created    int
	Privacy    int
	RenderHint int
}

type TitleRecord struct {
	Id    int
	Title string
}

func CreateAuthor(db *sql.DB, authorName string, password string) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return 0, err
	}
	query := "INSERT INTO users(userName, secret) VALUES(?, ?)"
	result, err := db.Exec(query, authorName, string(hashedPassword))
	if err != nil {
		return 0, err
	}
	lastRow, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	authorId := int(lastRow)

	query = "INSERT INTO sharing (user, sharesWith) VALUES (?, ?)"
	_, err = db.Exec(query, authorId, authorId)
	return authorId, err
}

func CreateNote(db *sql.DB, note *NoteRecord) (int, error) {
	query := "INSERT INTO notes(author, content, created, privacy, renderHint) " +
		"VALUES(?, ?, ?, ?, ?)"
	result, err := db.Exec(query, note.Author, note.Content, note.Created, note.Privacy, note.RenderHint)
	if err != nil {
		return 0, err
	}
	lastRow, err := result.LastInsertId()
	return int(lastRow), err
}

func CreateNoteDb(dbFileName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		return nil, err
	}

	queries := []string{
		"CREATE TABLE IF NOT EXISTS notes (author INT, content TEXT, created INT, privacy INT, renderHint INT)",
		"CREATE TABLE IF NOT EXISTS users (userName TEXT, secret TEXT)",
		"CREATE TABLE IF NOT EXISTS sharing (user INT, sharesWith INT, UNIQUE(user, sharesWith))",
		"CREATE INDEX IF NOT EXISTS idx_shares_with ON sharing (sharesWith)",
		"CREATE INDEX IF NOT EXISTS idx_sharing_users ON sharing (user)",
	}
	for _, query := range queries {
		if _, err = db.Exec(query); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func GetNote(db *sql.DB, userId int, noteId int) (*NoteRecord, error) {
	var note NoteRecord
	rows, err := db.Query(
		"SELECT author, content, created, IFNULL(privacy,0), IFNULL(renderHint,0) FROM notes, sharing "+
			"WHERE notes.rowId = ? AND ("+
			"notes.author = ? OR notes.privacy = ? OR "+
			"(notes.privacy = ? AND sharing.user = notes.author AND sharing.sharesWith = ?))",
		noteId, userId, PUBLIC_ACCESS, PROTECTED_ACCESS, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	if err = rows.Scan(&note.Author, &note.Content, &note.Created, &note.Privacy, &note.RenderHint); err != nil {
		return nil, err
	}
	return &note, nil
}

func GetTitles(db *sql.DB, userId int, noteIds []int) ([]TitleRecord, error) {
	rows, err := db.Query(
		"SELECT notes.rowId, note.content FROM notes, sharing "+
			"WHERE notes.rowId in ? AND ("+
			"notes.author = ? OR notes.privacy = ? OR "+
			"(notes.privacy = ? AND sharing.user = notes.author AND sharing.sharesWith = ?))",
		noteIds, userId, PUBLIC_ACCESS, PROTECTED_ACCESS, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []TitleRecord
	var titleRecord TitleRecord
	for rows.Next() {
		if err := rows.Scan(&titleRecord.Id, &titleRecord.Title); err != nil {
			return nil, err
		}
		titleRecord.Title = trimContentToTitle(titleRecord.Title)
		titles = append(titles, titleRecord)
	}
	return titles, nil
}

func SharesWith(db *sql.DB, sharerId int, shareeId int) error {
	query := "INSERT INTO sharing (user, sharesWith) VALUES (?, ?)"
	_, err := db.Exec(query, sharerId, shareeId)
	return err
}

// XXX fix!
func trimContentToTitle(content string) string {
	lines := strings.Split(content, "\n")
	return strings.Replace(lines[0], "#", "", 1)
}
