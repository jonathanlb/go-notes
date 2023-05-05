package index

import (
	"database/sql"
	"log"
	"org/bredin/go-notes/pkg/notes"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/mattn/go-sqlite3"
)

type SearchHit struct {
	Id    string
	Score float64
}

/**
 * Indexes about 40 documents/second on M1 Mac Mini....
 */
func CreateIndex(dbFileName string, indexDirName string) (bleve.Index, error) {
	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		return nil, err
	}

	defer func(db *sql.DB) {
		db.Close()
	}(db)

	indexMapping := bleve.NewIndexMapping()
	index, err := bleve.New(indexDirName, indexMapping)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT rowId, author, content, created FROM notes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type noteRecord struct {
		Author  int
		Content string
		Created int64
		Id      int
	}
	type noteIndex struct {
		Author  string
		Content string
		Created time.Time
		Id      string
	}
	note := noteRecord{}
	noteI := noteIndex{}
	for rows.Next() {
		if err := rows.Scan(&note.Id, &note.Author, &note.Content, &note.Created); err != nil {
			return nil, err
		}

		noteI.Id = strconv.Itoa(note.Id)
		noteI.Content = note.Content
		noteI.Created = time.Unix(note.Created, 0)

		author, err := notes.GetAuthor(db, note.Author)
		if err != nil {
			log.Fatalf("Cannot find author %d: %s", note.Author, err.Error())
		}
		noteI.Author = author.Name

		if err := index.Index(noteI.Id, noteI); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return index, nil
}

func OpenIndex(indexFileName string) (bleve.Index, error) {
	return bleve.Open(indexFileName)
}

func SearchIndex(index *bleve.Index, searchStr string) ([]SearchHit, error) {
	query := bleve.NewQueryStringQuery(searchStr)
	searchRequest := bleve.NewSearchRequest(query)
	searchResult, err := (*index).Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// TODO: trim for readability
	// how to avoid non-readable documents crowding out results?
	// Require +(Author:<user-id> Author:<shares-user>) ?
	var searchHits []SearchHit
	for _, h := range searchResult.Hits {
		searchHits = append(searchHits, SearchHit{h.ID, h.Score})
	}
	return searchHits, nil
}
